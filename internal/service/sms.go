package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
)

// SMSAPIURL is the nationalbulksms send endpoint.
const SMSAPIURL = "https://sms.nationalbulksms.com/fe/api/v1/send"

// otpRecord is one pending code.
type otpRecord struct {
	otp       string
	expiresAt time.Time
	// attempts counts WRONG guesses. Without it a six-digit code is only
	// single-use on the success path: a failed guess left the record intact, so
	// the 10-minute TTL was an unlimited guessing budget over a 900,000-value
	// space — exhaustible in minutes at a modest request rate.
	attempts int
}

// maskPhone renders a number for logs as 91XXXXXX8899: enough to correlate a
// support report, not enough to be a phone list if the log leaks.
func maskPhone(e164 string) string {
	if len(e164) <= 6 {
		return strings.Repeat("X", len(e164))
	}
	return e164[:2] + strings.Repeat("X", len(e164)-6) + e164[len(e164)-4:]
}

// SMSService generates, sends and verifies OTPs (§8.5).
type SMSService struct {
	cfg  *config.Config
	log  *slog.Logger
	http *http.Client

	mu    sync.RWMutex
	store map[string]otpRecord

	stop chan struct{}
}

// NewSMSService returns a service with a janitor goroutine running.
func NewSMSService(cfg *config.Config, log *slog.Logger) *SMSService {
	s := &SMSService{
		cfg:   cfg,
		log:   log,
		http:  &http.Client{Timeout: 10 * time.Second},
		store: make(map[string]otpRecord),
		stop:  make(chan struct{}),
	}
	go s.janitor()
	return s
}

// Close stops the janitor.
func (s *SMSService) Close() { close(s.stop) }

// janitor evicts expired codes.
//
// K8: the Python store never evicts, so it grows without bound. Adding a
// janitor is invisible to the contract because the EXPIRY semantics are
// unchanged — an expired code was already rejected, it just lingered in memory.
func (s *SMSService) janitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case now := <-ticker.C:
			s.mu.Lock()
			for k, r := range s.store {
				if now.After(r.expiresAt) {
					delete(s.store, k)
				}
			}
			s.mu.Unlock()
		}
	}
}

// E164 normalises an Indian mobile number to E.164 without the leading plus.
//
// Ported exactly, including the pass-through for anything that is neither 10
// nor 12 digits.
func E164(phone string) string {
	digits := strings.NewReplacer(" ", "", "+", "", "-", "").Replace(phone)
	if strings.HasPrefix(digits, "91") && len(digits) == 12 {
		return digits
	}
	if len(digits) == 10 {
		return "91" + digits
	}
	return digits
}

// generateOTP returns a 6-digit code in [100000, 999999].
//
// crypto/rand replaces Python's `random`, which is not seeded for security.
// §8.5 explicitly permits this: it is invisible to the contract and strictly
// safer, since a predictable OTP is a real account-takeover vector.
func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}

// SendOTP generates a code, sends it, and stores it with a 10-minute TTL.
//
// A non-2xx from the gateway is an error, which the handler surfaces as 502.
func (s *SMSService) SendOTP(ctx context.Context, phone string) error {
	otp, err := generateOTP()
	if err != nil {
		return fmt.Errorf("could not generate OTP: %w", err)
	}
	e164 := E164(phone)

	// KNOWN-ISSUE (K5): the brand in this text is "LaundryLy", a copy-paste
	// artefact from another project. It is DLT-registered per content in India,
	// so changing the wording silently breaks delivery until the template is
	// re-registered. Ported verbatim; product owns the fix.
	text := fmt.Sprintf(
		"Dear Customer, your LaundryLy verification OTP is %s. "+
			"It is valid for 10 minutes. Do not share this OTP with anyone. - Regards LaundryLy",
		otp)

	q := url.Values{}
	q.Set("username", s.cfg.SMSUsername)
	q.Set("password", s.cfg.SMSPassword)
	q.Set("unicode", "false")
	q.Set("from", s.cfg.SMSFrom)
	q.Set("to", e164)
	q.Set("text", text)
	q.Set("dltContentId", s.cfg.SMSDLTContentID)
	q.Set("dltPeid", s.cfg.SMSDLTPEID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, SMSAPIURL+"?"+q.Encode(), nil)
	if err != nil {
		return fmt.Errorf("Could not reach SMS gateway. Try again.")
	}
	resp, err := s.http.Do(req)
	if err != nil {
		s.log.Error("SMS send failed: " + err.Error())
		return fmt.Errorf("Could not reach SMS gateway. Try again.")
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 120))
	s.log.Info(fmt.Sprintf("SMS API response [%d]: %s", resp.StatusCode, string(body)))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("SMS gateway error %d: %s", resp.StatusCode, string(body))
	}

	s.mu.Lock()
	s.store[e164] = otpRecord{otp: otp, expiresAt: time.Now().Add(s.cfg.OTPExpiry)}
	s.mu.Unlock()

	s.log.Info(fmt.Sprintf("OTP stored for %s (expires in %ds)",
		maskPhone(e164), int(s.cfg.OTPExpiry.Seconds())))
	return nil
}

// VerifyOTP reports whether the code matches and has not expired.
//
// Single-use: a successful match deletes the record, and so does an expired
// one. Both behaviours are preserved from the Python.
//
// A wrong guess now counts against OTP_MAX_ATTEMPTS and discards the code once
// the budget is spent, so the code cannot be brute-forced inside its TTL. The
// RESPONSE is unchanged either way — false is false — so the frozen contract
// still holds; the caller simply has to request a new code after five misses.
func (s *SMSService) VerifyOTP(phone, otp string) bool {
	e164 := E164(phone)

	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.store[e164]
	if !ok {
		return false
	}
	if time.Now().After(rec.expiresAt) {
		delete(s.store, e164)
		return false
	}
	// Constant-time: the codes are short and single-use, so a timing oracle is
	// not the practical attack here, but there is no reason to leak the prefix.
	if subtle.ConstantTimeCompare([]byte(rec.otp), []byte(strings.TrimSpace(otp))) != 1 {
		rec.attempts++
		if rec.attempts >= s.maxAttempts() {
			delete(s.store, e164)
			s.log.Warn(fmt.Sprintf(
				"OTP for %s discarded after %d failed attempts", maskPhone(e164), rec.attempts))
		} else {
			s.store[e164] = rec
		}
		return false
	}
	delete(s.store, e164)
	return true
}

func (s *SMSService) maxAttempts() int {
	if s.cfg != nil && s.cfg.OTPMaxAttempts > 0 {
		return s.cfg.OTPMaxAttempts
	}
	return 5
}

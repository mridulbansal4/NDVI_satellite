package service

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
)

func testSMS(t *testing.T, maxAttempts int) *SMSService {
	t.Helper()
	cfg := &config.Config{OTPExpiry: 10 * time.Minute, OTPMaxAttempts: maxAttempts}
	s := NewSMSService(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(s.Close)
	return s
}

// seed injects a known code, so the tests do not need the SMS gateway.
func seed(s *SMSService, phone, otp string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[E164(phone)] = otpRecord{otp: otp, expiresAt: time.Now().Add(ttl)}
}

func TestVerifyOTPAcceptsTheCorrectCodeExactlyOnce(t *testing.T) {
	s := testSMS(t, 5)
	seed(s, "9876543210", "123456", time.Minute)

	if !s.VerifyOTP("9876543210", "123456") {
		t.Fatal("the correct code should verify")
	}
	if s.VerifyOTP("9876543210", "123456") {
		t.Fatal("a code must be single-use")
	}
}

func TestVerifyOTPDiscardsCodeAfterMaxFailedAttempts(t *testing.T) {
	s := testSMS(t, 3)
	seed(s, "9876543210", "123456", time.Minute)

	for i := 0; i < 3; i++ {
		if s.VerifyOTP("9876543210", "000000") {
			t.Fatalf("wrong code should not verify (attempt %d)", i+1)
		}
	}
	// The budget is spent: even the RIGHT code is now refused, because the
	// record was discarded. This is what stops the 900k-value brute force.
	if s.VerifyOTP("9876543210", "123456") {
		t.Fatal("the code should have been discarded after the attempt budget was spent")
	}
}

func TestVerifyOTPStillWorksWithinTheAttemptBudget(t *testing.T) {
	s := testSMS(t, 5)
	seed(s, "9876543210", "123456", time.Minute)

	for i := 0; i < 4; i++ {
		s.VerifyOTP("9876543210", "999999")
	}
	if !s.VerifyOTP("9876543210", "123456") {
		t.Fatal("a genuine user who mistypes a few times must still be able to verify")
	}
}

func TestVerifyOTPRejectsExpiredCode(t *testing.T) {
	s := testSMS(t, 5)
	seed(s, "9876543210", "123456", -time.Second) // already expired

	if s.VerifyOTP("9876543210", "123456") {
		t.Fatal("an expired code must not verify")
	}
	if _, present := s.store[E164("9876543210")]; present {
		t.Error("an expired record should be dropped on read")
	}
}

func TestVerifyOTPTrimsWhitespace(t *testing.T) {
	s := testSMS(t, 5)
	seed(s, "9876543210", "123456", time.Minute)
	if !s.VerifyOTP("9876543210", "  123456  ") {
		t.Fatal("surrounding whitespace should be tolerated, as in the Python")
	}
}

func TestVerifyOTPUnknownPhone(t *testing.T) {
	s := testSMS(t, 5)
	if s.VerifyOTP("9876543210", "123456") {
		t.Fatal("a phone with no pending code must not verify")
	}
}

func TestE164Normalisation(t *testing.T) {
	cases := map[string]string{
		"9876543210":    "919876543210",
		"+91 98765 43210": "919876543210",
		"919876543210":  "919876543210",
		"91-98765-43210": "919876543210",
		// Neither 10 nor 12 digits: passed through, as the Python does.
		"12345": "12345",
	}
	for in, want := range cases {
		if got := E164(in); got != want {
			t.Errorf("E164(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMaskPhoneKeepsCorrelationWithoutTheNumber(t *testing.T) {
	got := maskPhone("919876543210")
	if got == "919876543210" {
		t.Fatal("the number must not survive masking")
	}
	if want := "91XXXXXX3210"; got != want {
		t.Fatalf("maskPhone: got %q, want %q", got, want)
	}
	// Short input must not panic or slice out of range.
	if got := maskPhone("123"); got != "XXX" {
		t.Errorf("maskPhone(short): got %q, want XXX", got)
	}
	if got := maskPhone(""); got != "" {
		t.Errorf("maskPhone(empty): got %q", got)
	}
}

func TestGenerateOTPIsSixDigitsInRange(t *testing.T) {
	for i := 0; i < 200; i++ {
		otp, err := generateOTP()
		if err != nil {
			t.Fatalf("generateOTP: %v", err)
		}
		if len(otp) != 6 {
			t.Fatalf("OTP %q is not 6 characters", otp)
		}
		for _, r := range otp {
			if r < '0' || r > '9' {
				t.Fatalf("OTP %q contains a non-digit", otp)
			}
		}
		if otp[0] == '0' {
			t.Fatalf("OTP %q has a leading zero; range should be 100000-999999", otp)
		}
	}
}

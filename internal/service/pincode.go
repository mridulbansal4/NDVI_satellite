package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PinCodeAPIBase is the India Post lookup endpoint.
const PinCodeAPIBase = "https://api.postalpincode.in/pincode"

// pincodeUserAgent is the K11 fix, approved by the owner.
//
// api.postalpincode.in TCP-resets the default `python-requests/x.y.z`
// User-Agent, which is why GET /farmer/pincode/<pin> returns 500 on every call
// in the Python backend and POST /farmer/location silently stores empty
// state/district/taluka. Go's default is `Go-http-client/1.1`, which is not
// blocked — but relying on that would be luck, so an explicit UA is set.
//
// This is the ONE sanctioned deviation from §0.8 (preserve known bugs): the
// owner accepted the working behaviour as the new baseline, so the two e15_*
// goldens (which captured the broken 500) are excluded from the parity gate.
const pincodeUserAgent = "MindstriX-Backend/2.0 (+https://github.com/SanTiwari07/NDVI_satellite)"

// PinCodeResult is the resolved administrative area.
type PinCodeResult struct {
	State    string `json:"state"`
	District string `json:"district"`
	Taluka   string `json:"taluka"`
}

// PinCodeError distinguishes "not found" (→ 404) from "unreachable" (→ 500),
// mirroring the ValueError / RuntimeError split in utils/pincode.py.
type PinCodeError struct {
	Msg      string
	NotFound bool
}

func (e *PinCodeError) Error() string { return e.Msg }

// PinCodeClient calls the India Post API.
type PinCodeClient struct {
	Base string
	HTTP *http.Client
}

// NewPinCodeClient returns a client with the Python's 4-second timeout.
func NewPinCodeClient() *PinCodeClient {
	return &PinCodeClient{
		Base: PinCodeAPIBase,
		HTTP: &http.Client{Timeout: 4 * time.Second},
	}
}

// Resolve looks up state / district / taluka for a 6-digit PIN.
func (c *PinCodeClient) Resolve(ctx context.Context, pin string) (*PinCodeResult, error) {
	if len(pin) != 6 || !allDigits(pin) {
		return nil, &PinCodeError{
			Msg:      fmt.Sprintf("Invalid PIN code format: '%s'. Must be exactly 6 digits.", pin),
			NotFound: true,
		}
	}

	base := c.Base
	if base == "" {
		base = PinCodeAPIBase
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		base+"/"+url.PathEscape(pin), nil)
	if err != nil {
		return nil, &PinCodeError{Msg: "PIN code API unreachable: " + err.Error()}
	}
	req.Header.Set("User-Agent", pincodeUserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, &PinCodeError{Msg: "PIN code API unreachable: " + err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &PinCodeError{
			Msg: fmt.Sprintf("PIN code API unreachable: status %d", resp.StatusCode),
		}
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &PinCodeError{Msg: "PIN code API unreachable: " + err.Error()}
	}

	// The API answers with a single-element array.
	var payload []struct {
		Status     string `json:"Status"`
		PostOffice []struct {
			State    string `json:"State"`
			District string `json:"District"`
			Block    string `json:"Block"`
		} `json:"PostOffice"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &PinCodeError{Msg: "PIN code API unreachable: " + err.Error()}
	}

	if len(payload) == 0 || payload[0].Status != "Success" {
		return nil, &PinCodeError{
			Msg:      fmt.Sprintf("PIN code %s not found or invalid.", pin),
			NotFound: true,
		}
	}
	if len(payload[0].PostOffice) == 0 {
		return nil, &PinCodeError{
			Msg:      fmt.Sprintf("No post office data found for PIN code %s.", pin),
			NotFound: true,
		}
	}

	po := payload[0].PostOffice[0]
	// "Block" is the India Post field that corresponds to Taluka.
	return &PinCodeResult{State: po.State, District: po.District, Taluka: po.Block}, nil
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	return strings.IndexFunc(s, func(r rune) bool { return r < '0' || r > '9' }) < 0
}

package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
	"github.com/SanTiwari07/NDVI_satellite/internal/logging"
)

func newTestServer(t *testing.T) (http.Handler, *atomic.Bool, *atomic.Bool) {
	t.Helper()
	cfg, err := config.Load("\x00nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	cfg.GEEProjectID = "test-project"
	cfg.OllamaModel = "llama3"
	cfg.OllamaBaseURL = "http://127.0.0.1:11434"

	gee, fb := &atomic.Bool{}, &atomic.Bool{}
	h := New(Deps{
		Cfg:           cfg,
		Log:           logging.New("ERROR", ""),
		GEEReady:      gee,
		FirebaseReady: fb,
	})
	return h, gee, fb
}

func do(t *testing.T, h http.Handler, method, path, body string) (int, map[string]any) {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

// TestHealth pins §10.2: always 200, exactly four keys, and the two capability
// flags genuinely reflect the atomics rather than being hardcoded. The contract
// runner treats those flags as volatile, so this is where they are covered.
func TestHealth(t *testing.T) {
	h, gee, fb := newTestServer(t)

	code, body := do(t, h, "GET", "/health", "")
	if code != http.StatusOK {
		t.Errorf("status %d, want 200", code)
	}
	got := make([]string, 0, len(body))
	for k := range body {
		got = append(got, k)
	}
	sort.Strings(got)
	want := []string{"firebase_ready", "gee_ready", "project", "status"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("keys = %v, want %v", got, want)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %v, want ok", body["status"])
	}
	if body["gee_ready"] != false || body["firebase_ready"] != false {
		t.Errorf("flags should start false, got %v/%v", body["gee_ready"], body["firebase_ready"])
	}

	gee.Store(true)
	fb.Store(true)
	_, body = do(t, h, "GET", "/health", "")
	if body["gee_ready"] != true || body["firebase_ready"] != true {
		t.Errorf("flags did not track the atomics: %v/%v",
			body["gee_ready"], body["firebase_ready"])
	}
	if body["project"] != "test-project" {
		t.Errorf("project = %v", body["project"])
	}
}

// TestGEENotReadyWordings pins the detail §10.4 flags explicitly: E2 is the
// ONLY route using the long message. Unifying them would be a contract break.
func TestGEENotReadyWordings(t *testing.T) {
	h, _, _ := newTestServer(t)

	code, body := do(t, h, "POST", "/api/analyze", `{}`)
	if code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, want 503", code)
	}
	if body["error"] != msgGEENotReadyLong {
		t.Errorf("/api/analyze error = %q, want the LONG wording", body["error"])
	}

	for _, path := range []string{
		"/api/analyze-dates", "/api/analyze-day",
		"/api/analyze-radar-dates", "/api/analyze-radar",
	} {
		code, body := do(t, h, "POST", path, `{}`)
		if code != http.StatusServiceUnavailable {
			t.Errorf("%s: status %d, want 503", path, code)
		}
		if body["error"] != msgGEENotReadyShort {
			t.Errorf("%s: error = %q, want the SHORT wording", path, body["error"])
		}
	}

	code, body = do(t, h, "GET", "/api/sample?lat=1&lng=1", "")
	if code != http.StatusServiceUnavailable || body["error"] != msgGEENotReadyShort {
		t.Errorf("/api/sample: %d %q", code, body["error"])
	}
}

// TestSampleCheckOrdering pins §10.7: readiness is checked before the cached
// image, which is checked before coordinate parsing. With GEE ready but nothing
// analysed, bad coordinates must still yield the 404, not a 400.
func TestSampleCheckOrdering(t *testing.T) {
	h, gee, _ := newTestServer(t)
	gee.Store(true)

	code, body := do(t, h, "GET", "/api/sample?lat=nonsense", "")
	if code != http.StatusNotFound {
		t.Errorf("status %d, want 404 (cache check precedes coord parsing)", code)
	}
	if body["error"] != "No analysis available. Run an analysis first." {
		t.Errorf("error = %q", body["error"])
	}
}

func TestBandRepr(t *testing.T) {
	// The message embeds Python's list repr verbatim, single quotes included.
	want := "['NDVI', 'EVI', 'SAVI', 'NDMI', 'NDWI', 'GNDVI', 'CVI']"
	if got := sampleBandsRepr(); got != want {
		t.Errorf("sampleBandsRepr() = %q, want %q", got, want)
	}
}

// TestValidationEnvelopes asserts the marshmallow wordings captured from the
// running Flask app, including the 422-vs-400 split (§6.2).
func TestValidationEnvelopes(t *testing.T) {
	h, _, _ := newTestServer(t)

	cases := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantErrors map[string]string
	}{
		{
			"signup uses 422", "POST", "/auth/signup",
			`{"mobile_number":"123","password":"secret1"}`,
			http.StatusUnprocessableEntity,
			map[string]string{"mobile_number": "Mobile number must be exactly 10 digits."},
		},
		{
			"signup short password", "POST", "/auth/signup",
			`{"mobile_number":"9000000001","password":"abc"}`,
			http.StatusUnprocessableEntity,
			map[string]string{"password": "Password must be at least 6 characters."},
		},
		{
			"login uses 422", "POST", "/auth/login",
			`{"mobile_number":"abc","password":"x"}`,
			http.StatusUnprocessableEntity,
			map[string]string{"mobile_number": "Mobile number must be exactly 10 digits."},
		},
		{
			"consent missing field", "POST", "/consent", `{}`,
			http.StatusBadRequest,
			map[string]string{"satellite_monitoring": "Missing data for required field."},
		},
		{
			"soil oneof", "POST", "/soil",
			`{"farm_id":"00000000-0000-0000-0000-000000000001","soil_type":"purple"}`,
			http.StatusBadRequest,
			map[string]string{"soil_type": "Must be one of: black, red, sandy, mixed, unknown."},
		},
		{
			"irrigation oneof", "POST", "/irrigation",
			`{"farm_id":"00000000-0000-0000-0000-000000000001","irrigation_type":"magic"}`,
			http.StatusBadRequest,
			map[string]string{
				"irrigation_type": "Must be one of: rainfed, borewell, canal, drip_irrigation, sprinkler.",
			},
		},
		{
			"present-but-empty is a length error, not a missing-field error",
			"POST", "/farmer/basic-details",
			`{"name":"","preferred_language":"klingon"}`,
			http.StatusBadRequest,
			map[string]string{
				"name":               "Length must be between 1 and 255.",
				"preferred_language": "Must be one of: english, hindi, marathi, others.",
			},
		},
		{
			"pin_code custom message", "POST", "/farmer/location",
			`{"pin_code":"12","village_name":"Testville"}`,
			http.StatusBadRequest,
			map[string]string{"pin_code": "pin_code must be exactly 6 digits."},
		},
		{
			"farm numeric ranges", "POST", "/farm",
			`{"farm_name":"","total_area":0,"area_unit":"bushels","land_ownership":"nope","latitude":200,"longitude":400}`,
			http.StatusBadRequest,
			map[string]string{
				"farm_name":      "Length must be between 1 and 255.",
				"total_area":     "Must be greater than or equal to 0.01.",
				"area_unit":      "Must be one of: acres, hectares.",
				"land_ownership": "Must be one of: own_land, leased_land, contract_farming.",
				"latitude":       "Must be greater than or equal to -90 and less than or equal to 90.",
				"longitude":      "Must be greater than or equal to -180 and less than or equal to 180.",
			},
		},
		{
			"crop missing required fields", "POST", "/crop",
			`{"crop_name":"Grapes","season":"monsoon"}`,
			http.StatusBadRequest,
			map[string]string{
				"farm_id":     "Missing data for required field.",
				"sowing_date": "Missing data for required field.",
				"season":      "Must be one of: kharif, rabi, zaid.",
			},
		},
	}

	// Protected routes need a verified identity; JWT verification itself is
	// covered by the contract runner's minted-token cases.
	token := testToken(t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if !strings.HasPrefix(tc.path, "/auth/") {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status %d, want %d (body %s)", w.Code, tc.wantStatus, w.Body.String())
			}
			var got struct {
				Errors map[string][]string `json:"errors"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode: %v (body %s)", err, w.Body.String())
			}
			if len(got.Errors) != len(tc.wantErrors) {
				t.Errorf("got %d fields, want %d: %v", len(got.Errors), len(tc.wantErrors), got.Errors)
			}
			for field, wantMsg := range tc.wantErrors {
				msgs, ok := got.Errors[field]
				if !ok {
					t.Errorf("no error reported for %q (got %v)", field, got.Errors)
					continue
				}
				if len(msgs) != 1 || msgs[0] != wantMsg {
					t.Errorf("%s = %v, want [%q]", field, msgs, wantMsg)
				}
			}
		})
	}
}

// TestRouteTableIsComplete guards against a route being dropped: all 24
// endpoints of §10.1 must be registered and must not 404.
func TestRouteTableIsComplete(t *testing.T) {
	h, _, _ := newTestServer(t)
	routes := []struct{ method, path string }{
		{"GET", "/health"},
		{"POST", "/api/analyze"}, {"POST", "/api/analyze-dates"},
		{"POST", "/api/analyze-day"}, {"POST", "/api/analyze-radar-dates"},
		{"POST", "/api/analyze-radar"}, {"GET", "/api/sample"},
		{"POST", "/api/auth/verify-token"}, {"POST", "/api/auth/send-otp"},
		{"POST", "/api/auth/verify-otp"},
		{"POST", "/auth/signup"}, {"POST", "/auth/login"},
		{"POST", "/farmer/basic-details"}, {"POST", "/farmer/location"},
		{"GET", "/farmer/pincode/422001"},
		{"POST", "/farm"}, {"POST", "/crop"}, {"POST", "/irrigation"},
		{"POST", "/soil"}, {"POST", "/consent"}, {"GET", "/dashboard"},
		{"POST", "/chatbot/chat"}, {"POST", "/chatbot/reset"},
		{"GET", "/chatbot/health"},
	}
	if len(routes) != 24 {
		t.Fatalf("route table lists %d routes, §10.1 defines 24", len(routes))
	}
	for _, r := range routes {
		req := httptest.NewRequest(r.method, r.path, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code == http.StatusNotFound {
			t.Errorf("%s %s is not registered", r.method, r.path)
		}
		if w.Code == http.StatusMovedPermanently || w.Code == http.StatusTemporaryRedirect {
			t.Errorf("%s %s redirects (%d) — the axios client would lose its POST body",
				r.method, r.path, w.Code)
		}
	}
}

var _ = slog.LevelInfo

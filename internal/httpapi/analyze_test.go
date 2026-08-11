package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAnalyzeErrorBranchesAgainstGoldens closes the gap the contract runner
// cannot reach in this phase.
//
// In app.py the GEE-readiness check comes FIRST, so while gee_ready is false
// every analysis request short-circuits to 503 and the 400 branches are
// unreachable over HTTP. The Phase 1 exit gate nevertheless requires those
// branches to be verified, so this test flips the readiness flag and drives the
// handlers directly, comparing against the same goldens the runner uses.
func TestAnalyzeErrorBranchesAgainstGoldens(t *testing.T) {
	h, gee, _ := newTestServer(t)
	gee.Store(true) // pretend Phase 2 has landed; the pipeline still 501s

	type golden struct {
		Method   string          `json:"method"`
		Path     string          `json:"path"`
		Body     json.RawMessage `json:"body"`
		Status   int             `json:"status"`
		Response struct {
			Error string `json:"error"`
		} `json:"response"`
	}

	// Every non-503 error branch of E2-E6 that does not need a live pipeline.
	cases := []string{
		"e2_analyze_no_body",
		"e2_analyze_not_object",
		"e2_analyze_wrong_type",
		"e2_analyze_empty_coords",
		"e2_analyze_short_ring",
		"e2_analyze_bad_point",
		"e2_analyze_bad_lon",
		"e2_analyze_bad_lat",
		"e3_dates_no_body",
		"e3_dates_invalid",
		"e4_day_missing_date",
		"e4_day_missing_geometry",
		"e4_day_invalid_polygon",
		"e5_radar_dates_no_body",
		"e5_radar_dates_invalid",
		"e6_radar_no_body",
		"e6_radar_invalid",
	}

	root := repoRootT(t)
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, "testdata", "golden", name+".json"))
			if err != nil {
				t.Skipf("golden not present: %v", err)
			}
			var g golden
			if err := json.Unmarshal(raw, &g); err != nil {
				t.Fatalf("decode golden: %v", err)
			}

			req := httptest.NewRequest(g.Method, g.Path, strings.NewReader(string(g.Body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != g.Status {
				t.Errorf("status %d, want %d (body %s)", w.Code, g.Status, w.Body.String())
			}
			var got struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode response: %v (%s)", err, w.Body.String())
			}
			if got.Error != g.Response.Error {
				t.Errorf("error mismatch\n go: %q\n py: %q", got.Error, g.Response.Error)
			}
		})
	}
}

// TestAnalyzeDayRequiresBothFields pins the combined message: /api/analyze-day
// reports "Missing geometry or date" for either field, unlike every other
// analysis route.
func TestAnalyzeDayRequiresBothFields(t *testing.T) {
	h, gee, _ := newTestServer(t)
	gee.Store(true)

	for _, body := range []string{
		`{}`,
		`{"geometry":{"type":"Polygon","coordinates":[[[0,0],[1,0],[1,1],[0,0]]]}}`,
		`{"date":"2025-02-14"}`,
	} {
		code, out := do(t, h, "POST", "/api/analyze-day", body)
		if code != http.StatusBadRequest {
			t.Errorf("body %s: status %d, want 400", body, code)
		}
		if out["error"] != "Missing geometry or date" {
			t.Errorf("body %s: error = %q", body, out["error"])
		}
	}
}

// TestNonNumericCoordinateIs500 documents the parity behaviour for a coordinate
// that is present and correctly shaped but not numeric: validate_polygon raises
// TypeError, and app.py calls it outside its try block, so Flask 500s.
func TestNonNumericCoordinateIs500(t *testing.T) {
	h, gee, _ := newTestServer(t)
	gee.Store(true)

	code, _ := do(t, h, "POST", "/api/analyze",
		`{"geometry":{"type":"Polygon","coordinates":[[["a","b"],[1,0],[1,1],["a","b"]]]}}`)
	if code != http.StatusInternalServerError {
		t.Errorf("status %d, want 500 (Python raises TypeError here)", code)
	}
}

func repoRootT(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not locate repo root")
	return ""
}

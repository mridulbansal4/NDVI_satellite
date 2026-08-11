// Command contract is the Layer-2 HTTP contract runner.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §12.2.
//
// It replays the frozen corpus (captured alongside the goldens in
// testdata/golden/) against the Go backend and diffs each response against the
// response the Python backend gave for the identical request.
//
// Diff rules, per §12.2:
//   - status codes must be equal
//   - JSON must be deep-equal after sorting object keys, masking volatile
//     values (tile URLs, ids, tokens), and applying a 1e-6 float tolerance
//
// Exit code is non-zero if any case fails, so it can gate `make verify`.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
)

const floatTolerance = 1e-6

// volatileKeys hold values that legitimately differ between two runs and are
// compared for presence and type only.
var volatileKeys = map[string]bool{
	"ndvi_tile_url": true, "tile_url": true, "cvi_tile_url": true,
	"evi_tile_url": true, "savi_tile_url": true, "ndmi_tile_url": true,
	"ndwi_tile_url": true, "gndvi_tile_url": true, "smi_tile_url": true,
	"rvi_tile_url": true, "ratio_tile_url": true, "vv_tile_url": true,
	"vh_tile_url": true,
	"session_id":  true, "consent_id": true, "farmer_id": true,
	"token": true, "jti": true, "id": true, "farm_id": true,
	"created_at": true, "uid": true,

	// /health capability flags. These describe the runtime state of whichever
	// process is answering — whether THAT process reached Earth Engine and
	// Firebase — not the response contract. Comparing them between two
	// separate processes is meaningless (and Firebase in particular depends on
	// serviceAccountKey.json being present). The key set, the types and the
	// "status" value are still compared strictly, and internal/httpapi's
	// TestHealth asserts that each flag really does track its atomic.
	"gee_ready": true, "firebase_ready": true,
}

// notYetImplemented marks cases whose handler is still a Phase-N stub. They are
// reported but do not fail the run, so each phase's gate is only about what
// that phase claims to deliver.
const stubStatus = http.StatusNotImplemented

// geeNotReadyMessages are the two distinct wordings app.py uses when Earth
// Engine is unavailable. A Go 503 carrying one of them, where the golden
// expected something else, means "GEE is not wired in this phase" rather than
// "the handler is wrong" — the same category as a 501 stub. Once Phase 2 lands
// and gee_ready flips true, these cases start being compared for real without
// any change to the runner.
var geeNotReadyMessages = map[string]bool{
	"Google Earth Engine is not initialised. Check server logs.": true,
	"GEE not initialised": true,
}

// environmentDependent cases cannot be compared against their golden. See
// docs/KNOWN_ISSUES.md K11: the goldens captured a 500 caused by the India Post
// API blocking python-requests' User-Agent, and the owner accepted Go's working
// behaviour as the new baseline.
var environmentDependent = map[string]string{
	"e15_pincode_success":  "K11 — golden captured a UA-blocked 500",
	"e15_pincode_notfound": "K11 — golden captured a UA-blocked 500",
}

type golden struct {
	ID       string            `json:"id"`
	Method   string            `json:"method"`
	Path     string            `json:"path"`
	Query    map[string]string `json:"query"`
	Body     json.RawMessage   `json:"body"`
	Auth     string            `json:"auth"`
	Status   int               `json:"status"`
	Response json.RawMessage   `json:"response"`
	Note     string            `json:"note"`
}

type result struct {
	id      string
	state   string // PASS | FAIL | STUB | SKIP
	details []string
}

func main() {
	var (
		goBase     = flag.String("go-base", "http://127.0.0.1:5001", "Go backend base URL")
		pythonBase = flag.String("python-base", "", "if set, live-diff against this Python backend instead of the goldens")
		goldenDir  = flag.String("golden", "", "golden directory (default <repo>/testdata/golden)")
		only       = flag.String("only", "", "run only cases whose id contains this substring")
		verbose    = flag.Bool("v", false, "print the full diff for passing cases too")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(2)
	}
	jwtSecret = []byte(cfg.JWTSecret)

	dir := *goldenDir
	if dir == "" {
		root, err := repoRoot()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		dir = filepath.Join(root, "testdata", "golden")
	}

	goldens, err := loadGoldens(dir, *only)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if len(goldens) == 0 {
		fmt.Fprintf(os.Stderr, "no goldens found in %s — run `make fixtures-contract` first\n", dir)
		os.Exit(2)
	}

	client := &http.Client{Timeout: 300 * time.Second}
	var results []result
	var failed, stubbed, skipped int

	for _, g := range goldens {
		r := result{id: g.ID}

		if reason, ok := environmentDependent[g.ID]; ok {
			r.state = "SKIP"
			r.details = append(r.details, reason)
			results = append(results, r)
			skipped++
			continue
		}

		gotStatus, gotBody, err := fire(client, *goBase, g)
		if err != nil {
			r.state = "FAIL"
			r.details = append(r.details, "request failed: "+err.Error())
			results = append(results, r)
			failed++
			continue
		}

		wantStatus, wantBody := g.Status, g.Response
		if *pythonBase != "" {
			wantStatus, wantBody, err = fire(client, *pythonBase, g)
			if err != nil {
				r.state = "FAIL"
				r.details = append(r.details, "python request failed: "+err.Error())
				results = append(results, r)
				failed++
				continue
			}
		}

		if gotStatus == stubStatus && wantStatus != stubStatus {
			r.state = "STUB"
			r.details = append(r.details, stubReason(gotBody))
			results = append(results, r)
			stubbed++
			continue
		}

		if gotStatus == http.StatusServiceUnavailable &&
			wantStatus != http.StatusServiceUnavailable &&
			geeNotReadyMessages[stubReason(gotBody)] {
			r.state = "STUB"
			r.details = append(r.details, "GEE not wired in this phase: "+stubReason(gotBody))
			results = append(results, r)
			stubbed++
			continue
		}

		if gotStatus != wantStatus {
			r.state = "FAIL"
			r.details = append(r.details,
				fmt.Sprintf("status: got %d, want %d", gotStatus, wantStatus))
		}

		var gotV, wantV any
		_ = json.Unmarshal(gotBody, &gotV)
		_ = json.Unmarshal(wantBody, &wantV)
		if diffs := diff("", gotV, wantV); len(diffs) > 0 {
			r.state = "FAIL"
			r.details = append(r.details, diffs...)
		}

		if r.state == "" {
			r.state = "PASS"
		} else {
			failed++
		}
		results = append(results, r)
	}

	passed := 0
	for _, r := range results {
		switch r.state {
		case "PASS":
			passed++
			if !*verbose {
				continue
			}
		}
		fmt.Printf("%-5s %s\n", r.state, r.id)
		for _, d := range r.details {
			fmt.Printf("        %s\n", truncate(d, 300))
		}
	}

	fmt.Printf("\n%d cases: %d pass, %d fail, %d not-yet-implemented, %d skipped\n",
		len(results), passed, failed, stubbed, skipped)
	if failed > 0 {
		os.Exit(1)
	}
}

// contractFarmerID is a fixed identity for minted tokens. Phase 5 replaces it
// with the id returned by the setup signup case.
const contractFarmerID = "00000000-0000-0000-0000-0000000000c1"

var jwtSecret []byte

// mintToken produces a Flask-JWT-Extended-shaped access token (§8.3). The full
// claim set is emitted even though the verify path does not require it, so that
// a rollback to the Python backend keeps working.
func mintToken(expired bool, secret []byte) string {
	now := time.Now()
	exp := now.Add(7 * 24 * time.Hour)
	if expired {
		exp = now.Add(-time.Minute)
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   contractFarmerID,
		"type":  "access",
		"fresh": false,
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"exp":   exp.Unix(),
		"jti":   "contract-runner",
	})
	s, err := tok.SignedString(secret)
	if err != nil {
		return ""
	}
	return s
}

func stubReason(body []byte) string {
	var v struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(body, &v)
	if v.Error == "" {
		return "501 (handler not implemented yet)"
	}
	return v.Error
}

func fire(client *http.Client, base string, g golden) (int, []byte, error) {
	u := strings.TrimRight(base, "/") + g.Path
	if len(g.Query) > 0 {
		q := url.Values{}
		keys := make([]string, 0, len(g.Query))
		for k := range g.Query {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			q.Set(k, g.Query[k])
		}
		u += "?" + q.Encode()
	}

	var bodyReader io.Reader
	if g.Method == http.MethodPost && len(g.Body) > 0 && string(g.Body) != "null" {
		bodyReader = bytes.NewReader(g.Body)
	}

	req, err := http.NewRequest(g.Method, u, bodyReader)
	if err != nil {
		return 0, nil, err
	}
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	// Auth headers are MINTED per run, not replayed: a captured token would
	// have expired, and the "expired" case needs a token that is expired right
	// now. The secret comes from the same JWT_SECRET_KEY the servers read.
	switch g.Auth {
	case "malformed":
		req.Header.Set("Authorization", "not-a-bearer-token")
	case "valid":
		req.Header.Set("Authorization", "Bearer "+mintToken(false, jwtSecret))
	case "expired":
		req.Header.Set("Authorization", "Bearer "+mintToken(true, jwtSecret))
	case "badsig":
		req.Header.Set("Authorization", "Bearer "+mintToken(false, []byte("wrong-secret")))
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	return resp.StatusCode, raw, err
}

// diff walks two decoded JSON values and reports every structural or numeric
// difference. Object key order is irrelevant (Go maps are unordered anyway),
// floats compare within floatTolerance, and volatile keys compare on presence
// and JSON type only.
func diff(path string, got, want any) []string {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return []string{fmt.Sprintf("%s: got %T, want object", label(path), got)}
		}
		var out []string
		keys := make([]string, 0, len(w))
		for k := range w {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			gv, present := g[k]
			if !present {
				out = append(out, fmt.Sprintf("%s: missing key %q", label(path), k))
				continue
			}
			if volatileKeys[k] {
				if sameJSONKind(gv, w[k]) {
					continue
				}
				out = append(out, fmt.Sprintf("%s.%s: type differs (%T vs %T)", label(path), k, gv, w[k]))
				continue
			}
			out = append(out, diff(path+"."+k, gv, w[k])...)
		}
		for k := range g {
			if _, present := w[k]; !present {
				out = append(out, fmt.Sprintf("%s: unexpected key %q", label(path), k))
			}
		}
		return out

	case []any:
		g, ok := got.([]any)
		if !ok {
			return []string{fmt.Sprintf("%s: got %T, want array", label(path), got)}
		}
		if len(g) != len(w) {
			return []string{fmt.Sprintf("%s: length %d, want %d", label(path), len(g), len(w))}
		}
		var out []string
		for i := range w {
			out = append(out, diff(fmt.Sprintf("%s[%d]", path, i), g[i], w[i])...)
		}
		return out

	case float64:
		g, ok := got.(float64)
		if !ok {
			return []string{fmt.Sprintf("%s: got %v (%T), want number %v", label(path), got, got, w)}
		}
		if math.Abs(g-w) > floatTolerance {
			return []string{fmt.Sprintf("%s: %v, want %v (tolerance %g)", label(path), g, w, floatTolerance)}
		}
		return nil

	case nil:
		if got != nil {
			// This is the §13.1 trap: a masked cell must serialise as null,
			// never as 0. Call it out explicitly.
			return []string{fmt.Sprintf("%s: got %v, want null "+
				"(a null index value must not become 0 — see PRD §13.1)", label(path), got)}
		}
		return nil

	default:
		if fmt.Sprint(got) != fmt.Sprint(want) {
			return []string{fmt.Sprintf("%s: %q, want %q", label(path), fmt.Sprint(got), fmt.Sprint(want))}
		}
		return nil
	}
}

func sameJSONKind(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return fmt.Sprintf("%T", a) == fmt.Sprintf("%T", b)
}

func label(path string) string {
	if path == "" {
		return "$"
	}
	return "$" + path
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func loadGoldens(dir, only string) ([]golden, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []golden
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" || e.Name() == "manifest.json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var g golden
		if err := json.Unmarshal(raw, &g); err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if g.ID == "" || (only != "" && !strings.Contains(g.ID, only)) {
			continue
		}
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("could not locate repo root (no go.mod found)")
}

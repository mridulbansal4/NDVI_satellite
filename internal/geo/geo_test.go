package geo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestValidatePolygonAgainstGoldens replays the polygon-validation cases from
// the corpus and asserts the Go message matches, byte for byte, what the
// running Flask app returned. These strings reach the user, so wording,
// punctuation and number formatting are all part of the contract.
func TestValidatePolygonAgainstGoldens(t *testing.T) {
	type golden struct {
		Body     map[string]json.RawMessage `json:"body"`
		Status   int                        `json:"status"`
		Response struct {
			Error string `json:"error"`
		} `json:"response"`
	}

	cases := []string{
		"e2_analyze_not_object",
		"e2_analyze_wrong_type",
		"e2_analyze_empty_coords",
		"e2_analyze_short_ring",
		"e2_analyze_bad_point",
		"e2_analyze_bad_lon",
		"e2_analyze_bad_lat",
	}

	root := repoRoot(t)
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
			geometry, ok := g.Body["geometry"]
			if !ok {
				t.Fatalf("golden has no geometry in its request body")
			}
			err = ValidatePolygon(geometry)
			if err == nil {
				t.Fatalf("ValidatePolygon accepted a polygon Flask rejected with %q",
					g.Response.Error)
			}
			if err.Error() != g.Response.Error {
				t.Errorf("message mismatch\n go: %q\n py: %q", err.Error(), g.Response.Error)
			}
		})
	}
}

func TestValidatePolygonAcceptsFixtures(t *testing.T) {
	root := repoRoot(t)
	dir := filepath.Join(root, "testdata", "polygons")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("polygon fixtures not present: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidatePolygon(raw); err != nil {
				t.Errorf("fixture polygon rejected: %v", err)
			}
		})
	}
}

func TestValidatePolygonOrdering(t *testing.T) {
	// Check order matters: a geometry that fails several checks must report the
	// FIRST failure, matching Python's early returns.
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"type checked before coordinates",
			`{"type":"Point","coordinates":[]}`,
			"Geometry type must be 'Polygon', got 'Point'.",
		},
		{
			"missing type renders as None",
			`{"coordinates":[]}`,
			"Geometry type must be 'Polygon', got 'None'.",
		},
		{
			"coordinates not an array",
			`{"type":"Polygon","coordinates":"nope"}`,
			"Geometry 'coordinates' must be a non-empty array.",
		},
		{
			"ring arity checked before per-point checks",
			`{"type":"Polygon","coordinates":[[[999,999],[1,1],[2,2]]]}`,
			"Outer ring must have at least 4 coordinate pairs (3 unique + 1 closing).",
		},
		{
			"longitude checked before latitude on the same point",
			`{"type":"Polygon","coordinates":[[[181.0,91.0],[1,1],[2,2],[181.0,91.0]]]}`,
			"Longitude 181.0 at index 0 is out of range [-180, 180].",
		},
		{
			"holes are not validated",
			`{"type":"Polygon","coordinates":[[[0,0],[1,0],[1,1],[0,0]],[[999,999]]]}`,
			"",
		},
		{
			"integer literal keeps no decimal point",
			`{"type":"Polygon","coordinates":[[[200,0],[1,0],[1,1],[200,0]]]}`,
			"Longitude 200 at index 0 is out of range [-180, 180].",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePolygon(json.RawMessage(tc.in))
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNonNumericCoordinateIsSentinel(t *testing.T) {
	err := ValidatePolygon(json.RawMessage(
		`{"type":"Polygon","coordinates":[[["a","b"],[1,0],[1,1],["a","b"]]]}`))
	if !errors.Is(err, ErrNonNumericCoordinate) {
		t.Errorf("got %v, want ErrNonNumericCoordinate", err)
	}
}

// TestPyNumMatchesPython pins the formatting rules captured from the Python
// interpreter. PRD §6.1's suggested strconv.FormatFloat(v,'g',-1,64) fails the
// first four rows.
func TestPyNumMatchesPython(t *testing.T) {
	cases := []struct{ literal, want string }{
		{"181.0", "181.0"},
		{"91.5", "91.5"},
		{"-200.5", "-200.5"},
		{"2.5e3", "2500.0"},
		{"1e10", "10000000000.0"},
		{"1e16", "1e+16"},
		{"1e17", "1e+17"},
		{"1e20", "1e+20"},
		{"0.0001", "0.0001"},
		{"0.00001", "1e-05"},
		{"1e-7", "1e-07"},
		{"1.50", "1.5"},
		{"0.1", "0.1"},
		{"3.141592653589793", "3.141592653589793"},
		{"200", "200"},
		{"-200", "-200"},
		{"123456789012345678", "123456789012345678"},
		{"1e400", "inf"},
	}
	for _, tc := range cases {
		if got := pyNum(json.Number(tc.literal)); got != tc.want {
			t.Errorf("pyNum(%s) = %q, want %q", tc.literal, got, tc.want)
		}
	}
}

func repoRoot(t *testing.T) string {
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

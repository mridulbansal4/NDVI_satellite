package chatbot

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPromptMatchesPythonRender is the §10.11 acceptance test: render the Go
// template with a fixed input and diff it against the Python function's output
// for the same input, captured as a fixture.
//
// A single drifted character — a straight apostrophe instead of U+2019, a
// hyphen instead of the en dash — changes the model's context and would go
// unnoticed without this.
func TestPromptMatchesPythonRender(t *testing.T) {
	type fixtureCase struct {
		FarmData    map[string]any `json:"farm_data"`
		HeatmapData map[string]any `json:"heatmap_data"`
		Rendered    string         `json:"rendered"`
	}

	raw, err := os.ReadFile(filepath.Join("testdata", "system_prompt.json"))
	if err != nil {
		t.Skipf("fixture not present: %v", err)
	}
	// UseNumber so that 30 and 30.0 stay distinguishable, exactly as the
	// chatbot handler decodes the incoming farmData.
	var cases map[string]fixtureCase
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&cases); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("no fixture cases")
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			fd := farmFromMap(c.FarmData)
			hd := heatmapFromMap(c.HeatmapData)

			got, err := BuildSystemPrompt(fd, hd)
			if err != nil {
				t.Fatalf("BuildSystemPrompt: %v", err)
			}
			if got == c.Rendered {
				return
			}

			// Report the first differing line, with codepoints, since the
			// likely failures are invisible typography differences.
			gotLines := strings.Split(got, "\n")
			wantLines := strings.Split(c.Rendered, "\n")
			t.Errorf("render differs (go %d lines / %d chars, python %d lines / %d chars)",
				len(gotLines), len(got), len(wantLines), len(c.Rendered))
			for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
				g, w := "", ""
				if i < len(gotLines) {
					g = gotLines[i]
				}
				if i < len(wantLines) {
					w = wantLines[i]
				}
				if g != w {
					t.Errorf("line %d:\n  go: %q\n  py: %q\n  go runes: %v\n  py runes: %v",
						i+1, g, w, runesOf(g), runesOf(w))
					return
				}
			}
		})
	}
}

// TestPromptTypography guards the specific non-ASCII characters §10.11 calls
// out, independently of the fixture.
func TestPromptTypography(t *testing.T) {
	got, err := BuildSystemPrompt(FallbackFarm(), FallbackHeatmap())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		s    string
		what string
	}{
		{"farmer\u2019s field", "U+2019 right single quote"},
		{"\u2192", "U+2192 rightwards arrow"},
		{"2\u20133 days", "U+2013 en dash"},
		{"Do NOT use emojis.", "the no-emoji instruction"},
		{"==================================================", "the 50-char rules"},
	} {
		if !strings.Contains(got, want.s) {
			t.Errorf("rendered prompt is missing %s (%q)", want.what, want.s)
		}
	}
	if strings.HasPrefix(got, "\n") || strings.HasSuffix(got, "\n") {
		t.Error("prompt must be stripped at both ends, matching Python's .strip()")
	}
}

// TestIndexFormatting pins the {:.4f} rendering of the six index values.
func TestIndexFormatting(t *testing.T) {
	fd := FallbackFarm()
	fd.NDVI = 0.8651
	fd.CVI = 0.5
	fd.EVI = 1.0
	fd.Area = 2.5
	fd.Confidence = 0

	got, err := BuildSystemPrompt(fd, FallbackHeatmap())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"NDVI (Plant Greenness): 0.8651",
		"CVI (Overall Health Score): 0.5000",
		"EVI (Canopy Density): 1.0000",
		"Area: 2.5 hectares",
		// An integral value arriving as JSON 0 must print "0", not "0.0" —
		// Python's json.loads yields an int for that literal.
		"Engine Confidence: 0%",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func farmFromMap(m map[string]any) FarmData {
	return FarmData{
		FieldName: m["fieldName"], Area: m["area"], Date: m["date"],
		Confidence: m["confidence"], CleanScenes: m["cleanScenes"],
		CVI: m["cvi"], NDVI: m["ndvi"], EVI: m["evi"],
		SAVI: m["savi"], NDMI: m["ndmi"], GNDVI: m["gndvi"],
	}
}

func heatmapFromMap(m map[string]any) HeatmapData {
	return HeatmapData{
		StressedPct: m["stressedPct"], StressedLocation: m["stressedLocation"],
		ModeratePct: m["moderatePct"], ModerateLocation: m["moderateLocation"],
		HealthyPct: m["healthyPct"], HealthyLocation: m["healthyLocation"],
	}
}

func runesOf(s string) []string {
	out := []string{}
	for _, r := range s {
		if r > 127 {
			out = append(out, string(r)+"=U+"+strings.ToUpper(hex(r)))
		}
	}
	return out
}

func hex(r rune) string {
	const digits = "0123456789abcdef"
	if r == 0 {
		return "0"
	}
	var buf []byte
	for r > 0 {
		buf = append([]byte{digits[r%16]}, buf...)
		r /= 16
	}
	return string(buf)
}

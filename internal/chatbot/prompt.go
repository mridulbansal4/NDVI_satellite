// Package chatbot ports the Krishi Mitra prompt builder and session memory.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §10.11.
//
// The system prompt must render character-for-character identically to
// prompts/system_prompt.py, including the non-ASCII typography (the U+2019
// apostrophe in "farmer's", the U+2192 arrows, the U+2013 en dash in "2–3
// days") and the "Do NOT use emojis." line. prompt_test.go diffs the render
// against output captured from the Python function.
package chatbot

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
)

// FarmData mirrors the farmData object the frontend posts on every message.
//
// Every field is `any` because the frontend sends whatever it has: numbers,
// strings, or nothing at all. The Python formats them with plain f-string
// interpolation, so 0 renders "0" and 2.5 renders "2.5" — reproducing that
// requires seeing the original JSON type rather than coercing to float64.
type FarmData struct {
	FieldName   any `json:"fieldName"`
	Area        any `json:"area"`
	Date        any `json:"date"`
	Confidence  any `json:"confidence"`
	CleanScenes any `json:"cleanScenes"`
	CVI         any `json:"cvi"`
	NDVI        any `json:"ndvi"`
	EVI         any `json:"evi"`
	SAVI        any `json:"savi"`
	NDMI        any `json:"ndmi"`
	GNDVI       any `json:"gndvi"`
}

// HeatmapData mirrors the heatmapData object.
type HeatmapData struct {
	StressedPct      any `json:"stressedPct"`
	StressedLocation any `json:"stressedLocation"`
	ModeratePct      any `json:"moderatePct"`
	ModerateLocation any `json:"moderateLocation"`
	HealthyPct       any `json:"healthyPct"`
	HealthyLocation  any `json:"healthyLocation"`
}

// FallbackFarm mirrors _FALLBACK_FARM in chatbot/routes.py, used when the
// frontend omits farmData.
func FallbackFarm() FarmData {
	return FarmData{
		FieldName: "Unknown Field", Area: 0, Date: "Unknown",
		Confidence: 0, CleanScenes: 0,
		CVI: 0, NDVI: 0, EVI: 0, SAVI: 0, NDMI: 0, GNDVI: 0,
	}
}

// FallbackHeatmap mirrors _FALLBACK_HEATMAP.
func FallbackHeatmap() HeatmapData {
	return HeatmapData{
		StressedPct: 0, StressedLocation: "the field",
		ModeratePct: 0, ModerateLocation: "the field",
		HealthyPct: 0, HealthyLocation: "the field",
	}
}

// systemPromptTemplate is a character-for-character port. The `py` function
// reproduces Python's f-string rendering of an arbitrary value; `f4` is the
// {:.4f} format the six index values use.
var systemPromptTemplate = template.Must(
	template.New("system_prompt").
		Funcs(template.FuncMap{"py": pyFormat, "f4": format4}).
		Parse(rawSystemPrompt))

// BuildSystemPrompt renders the prompt for one request. The prompt is rebuilt
// per message because it embeds the current field's live statistics.
func BuildSystemPrompt(fd FarmData, hd HeatmapData) (string, error) {
	var sb strings.Builder
	err := systemPromptTemplate.Execute(&sb, map[string]any{"fd": fd, "hd": hd})
	if err != nil {
		return "", err
	}
	// The Python ends with .strip().
	return strings.TrimSpace(sb.String()), nil
}

// format4 reproduces Python's float(x) followed by "{:.4f}".
func format4(v any) string {
	f, ok := toFloat(v)
	if !ok {
		// float(x) on a non-numeric raises; the caller would 500. Rendering
		// the raw value keeps the failure visible rather than silently zeroing.
		return pyFormat(v)
	}
	return fmt.Sprintf("%.4f", f)
}

func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		var f float64
		if _, err := fmt.Sscanf(t, "%g", &f); err == nil {
			return f, true
		}
	}
	return 0, false
}

// pyFormat renders a decoded JSON value the way a Python f-string would.
//
// The subtlety is int-vs-float. json.loads keeps "30" as a Python int (prints
// "30") and "30.0" as a float (prints "30.0"), but Go's encoding/json collapses
// both to float64 and would print "30" for each. Decoding with UseNumber
// preserves the original literal, which is the only way to tell them apart —
// the same problem, and the same fix, as internal/geo.pyNum.
func pyFormat(v any) string {
	switch t := v.(type) {
	case nil:
		return "None"
	case bool:
		if t {
			return "True"
		}
		return "False"
	case string:
		return t
	case json.Number:
		return pyNumber(t)
	case float64:
		// Reached only when the caller did not use UseNumber. Treated as a
		// Python float, so an integral value keeps its ".0".
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d.0", int64(t))
		}
		return fmt.Sprintf("%g", t)
	case int:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	}
	return fmt.Sprintf("%v", v)
}

// pyNumber renders a JSON number literal as Python's str() would.
func pyNumber(n json.Number) string {
	s := n.String()
	if !strings.ContainsAny(s, ".eE") {
		return s // a Python int, printed verbatim
	}
	f, err := n.Float64()
	if err != nil {
		return s
	}
	// Python's float repr: shortest round trip, always at least one decimal.
	out := fmt.Sprintf("%v", f)
	if !strings.ContainsAny(out, ".eE") {
		out += ".0"
	}
	return out
}

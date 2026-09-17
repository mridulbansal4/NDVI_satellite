package config

import (
	"testing"
	"time"
)

// The literals below are a SECOND, INDEPENDENT transcription of
// backend/config.py, backend/chatbot/config.py and backend/app.py, made
// deliberately by re-reading the Python source rather than by copying
// config.go. A typo would have to be made identically twice to slip through.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §14 Phase 0, Appendix A.

func load(t *testing.T) *Config {
	t.Helper()
	// No .env files: assert the compiled-in defaults, not this machine's env.
	c, err := Load("\x00nonexistent")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return c
}

func TestEarthEngineConstants(t *testing.T) {
	c := load(t)
	if c.Dataset != "COPERNICUS/S2_SR_HARMONIZED" {
		t.Errorf("Dataset = %q", c.Dataset)
	}
	if c.LookbackDays != 90 {
		t.Errorf("LookbackDays = %d, want 90", c.LookbackDays)
	}
	if c.MaxCloudCoverPct != 20 {
		t.Errorf("MaxCloudCoverPct = %d, want 20", c.MaxCloudCoverPct)
	}
	wantSCL := []int{3, 8, 9, 10}
	if len(c.SCLMaskValues) != len(wantSCL) {
		t.Fatalf("SCLMaskValues = %v", c.SCLMaskValues)
	}
	for i, v := range wantSCL {
		if c.SCLMaskValues[i] != v {
			t.Errorf("SCLMaskValues[%d] = %d, want %d", i, c.SCLMaskValues[i], v)
		}
	}
	for band, want := range map[string]string{
		"BLUE": "B2", "GREEN": "B3", "RED": "B4",
		"NIR": "B8", "SWIR": "B11", "SCL": "SCL",
	} {
		if got := c.Bands[band]; got != want {
			t.Errorf("Bands[%q] = %q, want %q", band, got, want)
		}
	}
}

func TestCVIWeights(t *testing.T) {
	c := load(t)
	// config.py values. README.md and the vi_reports comment quote an older
	// 0.35/0.25/0.15/0.15/0.10 set — that is K2, and the code wins.
	want := map[string]float64{
		"NDVI": 0.70, "EVI": 0.10, "SAVI": 0.05, "NDMI": 0.10, "GNDVI": 0.05,
	}
	if len(c.CVIWeights) != len(want) {
		t.Fatalf("CVIWeights has %d entries, want %d (NDWI must NOT be weighted)",
			len(c.CVIWeights), len(want))
	}
	var sum float64
	for k, v := range want {
		if c.CVIWeights[k] != v {
			t.Errorf("CVIWeights[%q] = %v, want %v", k, c.CVIWeights[k], v)
		}
		sum += c.CVIWeights[k]
	}
	// Map iteration order is random and float addition is not associative, so
	// the sum can land on 0.9999999999999999. Same tolerance as Validate().
	if diff := sum - 1.0; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("CVI weights sum to %v, want 1.0", sum)
	}
	if _, ok := c.CVIWeights["NDWI"]; ok {
		t.Error("NDWI must not carry a CVI weight")
	}
}

func TestGridAndConfidenceConstants(t *testing.T) {
	c := load(t)
	if c.GridScaleM != 10 || c.MaxGridCells != 2000 || c.GridScaleStepM != 2 {
		t.Errorf("grid = %d/%d/%d, want 10/2000/2",
			c.GridScaleM, c.MaxGridCells, c.GridScaleStepM)
	}
	// Both call sites (vegetation and radar) pass 0.6 explicitly.
	if c.SmoothSigmaFactor != 0.6 {
		t.Errorf("SmoothSigmaFactor = %v, want 0.6", c.SmoothSigmaFactor)
	}
	if c.ConfidenceSceneTarget != 5 {
		t.Errorf("ConfidenceSceneTarget = %d, want 5", c.ConfidenceSceneTarget)
	}
	if c.ConfidenceStdMax != 0.3 {
		t.Errorf("ConfidenceStdMax = %v, want 0.3", c.ConfidenceStdMax)
	}
}

func TestThresholdTables(t *testing.T) {
	c := load(t)
	cases := []struct {
		name string
		got  []Threshold
		want []Threshold
	}{
		{"CVI", c.CVIThresholds, []Threshold{
			{0.5, "Healthy vegetation"},
			{0.25, "Moderate vegetation, possible stress"},
			{-1.0, "Poor vegetation, needs attention"},
		}},
		{"NDVI", c.NDVIThresholds, []Threshold{
			{0.6, "Dense, healthy vegetation"},
			{0.4, "Moderate vegetation"},
			{0.2, "Sparse / stressed vegetation"},
			{0.0, "Bare soil"},
			{-1.0, "Water / non-vegetated"},
		}},
		{"EVI", c.EVIThresholds, []Threshold{
			{0.5, "Dense vegetation"},
			{0.3, "Moderate vegetation"},
			{0.1, "Sparse vegetation"},
			{-1.0, "Bare / non-vegetated"},
		}},
		{"SAVI", c.SAVIThresholds, []Threshold{
			{0.5, "High vegetation + low soil effect"},
			{0.3, "Moderate vegetation"},
			{0.1, "Low vegetation"},
			{-1.0, "Bare soil dominant"},
		}},
		{"NDMI", c.NDMIThresholds, []Threshold{
			{0.4, "High moisture"},
			{0.2, "Moderate moisture"},
			{0.0, "Low moisture"},
			{-1.0, "Dry / drought stress"},
		}},
		{"NDWI", c.NDWIThresholds, []Threshold{
			{0.3, "High water presence"},
			{0.0, "Water likely present"},
			{-1.0, "No significant water"},
		}},
		{"GNDVI", c.GNDVIThresholds, []Threshold{
			{0.6, "Excellent chlorophyll / nutrient status"},
			{0.4, "Good chlorophyll"},
			{0.2, "Moderate"},
			{-1.0, "Low chlorophyll"},
		}},
		{"SMI", c.SMIThresholds, []Threshold{
			{0.66, "Wet"},
			{0.33, "Moderate"},
			{0.0, "Dry"},
		}},
	}
	for _, tc := range cases {
		if len(tc.got) != len(tc.want) {
			t.Errorf("%s: %d thresholds, want %d", tc.name, len(tc.got), len(tc.want))
			continue
		}
		for i := range tc.want {
			if tc.got[i] != tc.want[i] {
				t.Errorf("%s[%d] = %+v, want %+v", tc.name, i, tc.got[i], tc.want[i])
			}
		}
	}
}

// TestThresholdsAreSortedDescending is the guard §9.1 asks for: the tables
// replace config.py's sorted(keys, reverse=True) at every call site, so the
// ordering itself is load-bearing.
func TestThresholdsAreSortedDescending(t *testing.T) {
	c := load(t)
	for name, ts := range map[string][]Threshold{
		"CVI": c.CVIThresholds, "NDVI": c.NDVIThresholds, "EVI": c.EVIThresholds,
		"SAVI": c.SAVIThresholds, "NDMI": c.NDMIThresholds, "NDWI": c.NDWIThresholds,
		"GNDVI": c.GNDVIThresholds, "SMI": c.SMIThresholds,
	} {
		for i := 1; i < len(ts); i++ {
			if ts[i].Min >= ts[i-1].Min {
				t.Errorf("%s not descending at index %d: %v >= %v",
					name, i, ts[i].Min, ts[i-1].Min)
			}
		}
	}
}

func TestSentinel1Constants(t *testing.T) {
	c := load(t)
	if c.S1Dataset != "COPERNICUS/S1_GRD" {
		t.Errorf("S1Dataset = %q", c.S1Dataset)
	}
	if c.S1InstrumentMode != "IW" || c.S1OrbitPass != "DESCENDING" ||
		c.S1SpeckleKernel != "circle" {
		t.Errorf("S1 mode/orbit/kernel = %q/%q/%q",
			c.S1InstrumentMode, c.S1OrbitPass, c.S1SpeckleKernel)
	}
	if c.S1ResolutionM != 10 || c.S1LookbackDays != 90 ||
		c.S1DateWindowDays != 6 || c.S1SpeckleRadiusM != 50 {
		t.Errorf("S1 numerics = %d/%d/%d/%d, want 10/90/6/50",
			c.S1ResolutionM, c.S1LookbackDays, c.S1DateWindowDays, c.S1SpeckleRadiusM)
	}
	if c.SMIVVDryDB != -20.0 || c.SMIVVWetDB != -8.0 {
		t.Errorf("SMI bounds = %v/%v, want -20.0/-8.0", c.SMIVVDryDB, c.SMIVVWetDB)
	}
	wantBands := []string{"SMI", "RVI", "RATIO", "VV", "VH"}
	if len(c.RadarBands) != len(wantBands) {
		t.Fatalf("RadarBands = %v", c.RadarBands)
	}
	for i, b := range wantBands {
		if c.RadarBands[i] != b {
			t.Errorf("RadarBands[%d] = %q, want %q", i, c.RadarBands[i], b)
		}
	}
	for band, want := range map[string]VisBound{
		"SMI": {0.0, 1.0}, "RVI": {0.0, 1.0}, "RATIO": {2.0, 16.0},
		"VV": {-22.0, -6.0}, "VH": {-28.0, -12.0},
	} {
		if got := c.RadarVisBounds[band]; got != want {
			t.Errorf("RadarVisBounds[%q] = %+v, want %+v", band, got, want)
		}
	}
}

// TestPalettes is the typo guard §5.6 asks for. The '#' prefix is significant:
// the palette strings go into Image.visualize verbatim.
func TestPalettes(t *testing.T) {
	c := load(t)
	if len(c.NDVIPalette) != 21 {
		t.Errorf("NDVIPalette has %d colours, want 21", len(c.NDVIPalette))
	}
	if c.NDVIPalette[0] != "#ad0028" {
		t.Errorf("NDVIPalette[0] = %q, want %q", c.NDVIPalette[0], "#ad0028")
	}
	if last := c.NDVIPalette[len(c.NDVIPalette)-1]; last != "#007e47" {
		t.Errorf("NDVIPalette last = %q, want %q", last, "#007e47")
	}
	// The last two entries are deliberately identical in app.py.
	if c.NDVIPalette[19] != c.NDVIPalette[20] {
		t.Errorf("NDVIPalette[19] != [20]: %q vs %q", c.NDVIPalette[19], c.NDVIPalette[20])
	}

	wantCVI := []string{"#ef4444", "#f59e0b", "#22c55e"}
	if len(c.CVIPalette) != 3 {
		t.Fatalf("CVIPalette = %v", c.CVIPalette)
	}
	for i, want := range wantCVI {
		if c.CVIPalette[i] != want {
			t.Errorf("CVIPalette[%d] = %q, want %q", i, c.CVIPalette[i], want)
		}
	}

	if len(c.RadarMoisturePalette) != 9 {
		t.Errorf("RadarMoisturePalette has %d colours, want 9", len(c.RadarMoisturePalette))
	}
	if c.RadarMoisturePalette[0] != "#b30000" ||
		c.RadarMoisturePalette[8] != "#006837" {
		t.Errorf("RadarMoisturePalette ends = %q…%q",
			c.RadarMoisturePalette[0], c.RadarMoisturePalette[8])
	}
	if len(c.RadarSequentialPalette) != 5 {
		t.Errorf("RadarSequentialPalette has %d colours, want 5", len(c.RadarSequentialPalette))
	}
	if c.RadarSequentialPalette[0] != "#f7fbff" ||
		c.RadarSequentialPalette[4] != "#08519c" {
		t.Errorf("RadarSequentialPalette ends = %q…%q",
			c.RadarSequentialPalette[0], c.RadarSequentialPalette[4])
	}

	for name, pal := range map[string][]string{
		"NDVI": c.NDVIPalette, "CVI": c.CVIPalette,
		"RadarMoisture": c.RadarMoisturePalette,
		"RadarSeq":      c.RadarSequentialPalette,
	} {
		for i, col := range pal {
			if len(col) != 7 || col[0] != '#' {
				t.Errorf("%s[%d] = %q: want a 7-char '#rrggbb' literal", name, i, col)
			}
		}
	}
}

func TestIntegrationDefaults(t *testing.T) {
	c := load(t)
	if c.Port != 5000 {
		t.Errorf("Port = %d, want 5000 (FLASK_PORT name kept for ops continuity)", c.Port)
	}
	if c.JWTSecret != "dev-secret-change-me" {
		t.Errorf("JWTSecret default = %q", c.JWTSecret)
	}
	if c.JWTExpiry != 7*24*time.Hour {
		t.Errorf("JWTExpiry = %v, want 168h", c.JWTExpiry)
	}
	if c.OllamaTemperature != 0.7 {
		t.Errorf("OllamaTemperature = %v, want 0.7", c.OllamaTemperature)
	}
	if c.OllamaMaxTokens != 512 {
		t.Errorf("OllamaMaxTokens = %d, want 512", c.OllamaMaxTokens)
	}
	if c.ChatbotMaxHistory != 20 {
		t.Errorf("ChatbotMaxHistory = %d, want 20", c.ChatbotMaxHistory)
	}
	if c.OTPExpiry != 600*time.Second {
		t.Errorf("OTPExpiry = %v, want 600s", c.OTPExpiry)
	}
	if c.DBMinConns != 2 || c.DBMaxConns != 20 {
		t.Errorf("pool = %d/%d, want 2/20", c.DBMinConns, c.DBMaxConns)
	}
	if c.ExprCacheTTL != 30*time.Minute || c.ExprCacheMaxEntries != 256 {
		t.Errorf("expr cache = %v/%d, want 30m/256", c.ExprCacheTTL, c.ExprCacheMaxEntries)
	}
}

func TestCORSOrigins(t *testing.T) {
	c := load(t)
	want := []string{
		"http://localhost:5173",
		"http://localhost:5174",
		"http://localhost:5175",
		"http://localhost:4173",
		"http://localhost:3000",
	}
	if len(c.CORSOrigins) != len(want) {
		t.Fatalf("CORSOrigins = %v", c.CORSOrigins)
	}
	for i, o := range want {
		if c.CORSOrigins[i] != o {
			t.Errorf("CORSOrigins[%d] = %q, want %q", i, c.CORSOrigins[i], o)
		}
	}
}

func TestInterpret(t *testing.T) {
	c := load(t)
	f := func(v float64) *float64 { return &v }
	cases := []struct {
		v    *float64
		want string
	}{
		{nil, "N/A — data unavailable"}, // em dash is U+2014
		{f(0.9), "Dense, healthy vegetation"},
		{f(0.6), "Dense, healthy vegetation"}, // boundary is inclusive
		{f(0.5), "Moderate vegetation"},
		{f(0.0), "Bare soil"},
		{f(-0.5), "Water / non-vegetated"},
		{f(-2.0), "Unknown"}, // below every threshold
	}
	for _, tc := range cases {
		got := Interpret(tc.v, c.NDVIThresholds, "N/A — data unavailable", "Unknown")
		if got != tc.want {
			t.Errorf("Interpret(%v) = %q, want %q", tc.v, got, tc.want)
		}
	}
}

func TestValidateRejectsBadWeights(t *testing.T) {
	c := load(t)
	c.CVIWeights = map[string]float64{"NDVI": 0.5}
	if err := c.Validate(); err == nil {
		t.Error("Validate accepted weights that do not sum to 1.0")
	}
}

func TestValidateRejectsUnsortedThresholds(t *testing.T) {
	c := load(t)
	c.CVIThresholds = []Threshold{{0.1, "low"}, {0.9, "high"}}
	if err := c.Validate(); err == nil {
		t.Error("Validate accepted ascending thresholds")
	}
}

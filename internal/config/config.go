// Package config is the single tuning surface for the backend, mirroring
// backend/config.py and backend/chatbot/config.py 1:1.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §9.1, Appendix A.
//
// Rules (§0.4): no thresholds, weights, band names, dataset IDs or palettes may
// be hardcoded in service packages. Everything lives here and is passed
// explicitly to constructors — no package-level mutable globals, no init()
// side effects, and os.Getenv is read nowhere outside this package.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Threshold is one (minimum, label) pair in an interpretation table.
//
// config.py stores these as dicts and sorts the keys descending at every call
// site. A Go map has no order, so they are stored pre-sorted descending here
// and iterated directly. TestThresholdsAreSortedDescending guards the ordering
// so a future edit cannot silently break lookup.
type Threshold struct {
	Min   float64
	Label string
}

// VisBound is a per-band min/max pair used purely for colour mapping.
type VisBound struct {
	Min float64
	Max float64
}

// Config holds every tunable value. It is loaded once at startup and passed
// explicitly to the packages that need it.
type Config struct {
	// ── Server ──────────────────────────────────────────────────────────
	Port        int      // FLASK_PORT — env var name kept for ops continuity
	Env         string   // FLASK_ENV
	CORSOrigins []string // §9.3, hardcoded list

	// ── Earth Engine ────────────────────────────────────────────────────
	GEEProjectID         string // GEE_PROJECT_ID (required for gee_ready)
	GEEServiceAccountKey string // GEE_SERVICE_ACCOUNT_KEY, new in this migration
	GEEUserCredentials   string // GEE_USER_CREDENTIALS, dev fallback (§5.5)
	Dataset              string
	LookbackDays         int
	MaxCloudCoverPct     int
	SCLMaskValues        []int
	Bands                map[string]string

	// ── Indices / grid / confidence ─────────────────────────────────────
	CVIWeights            map[string]float64
	GridScaleM            int
	MaxGridCells          int
	GridScaleStepM        int
	SmoothSigmaFactor     float64
	ConfidenceSceneTarget int
	ConfidenceStdMax      float64

	// ── Interpretation thresholds (ordered descending) ──────────────────
	CVIThresholds   []Threshold
	NDVIThresholds  []Threshold
	EVIThresholds   []Threshold
	SAVIThresholds  []Threshold
	NDMIThresholds  []Threshold
	NDWIThresholds  []Threshold
	GNDVIThresholds []Threshold
	SMIThresholds   []Threshold

	// ── Sentinel-1 ──────────────────────────────────────────────────────
	S1Dataset        string
	S1InstrumentMode string
	S1OrbitPass      string
	S1SpeckleKernel  string
	S1ResolutionM    int
	S1LookbackDays   int
	S1DateWindowDays int
	S1SpeckleRadiusM int
	SMIVVDryDB       float64
	SMIVVWetDB       float64
	RadarBands       []string
	RadarVisBounds   map[string]VisBound

	// ── Palettes (from app.py) ──────────────────────────────────────────
	NDVIPalette            []string
	CVIPalette             []string
	RadarMoisturePalette   []string
	RadarSequentialPalette []string

	// ── Data stores / integrations ──────────────────────────────────────
	DatabaseURL       string
	FirebaseProjectID string
	ServiceAccountKey string // repo-root serviceAccountKey.json (§8.4)
	JWTSecret         string
	JWTExpiry         time.Duration
	OllamaBaseURL     string
	OllamaModel       string
	OllamaTemperature float64
	OllamaMaxTokens   int
	ChatbotMaxHistory int
	SMSUsername       string
	SMSPassword       string
	SMSFrom           string
	SMSDLTContentID   string
	SMSDLTPEID        string
	OTPExpiry         time.Duration

	// ── Expression cache (new, §7.7) ────────────────────────────────────
	ExprCacheTTL        time.Duration
	ExprCacheMaxEntries int

	// ── Database pool (§13.5) ───────────────────────────────────────────
	DBMinConns int32
	DBMaxConns int32

	// ── Logging (§9.4) ──────────────────────────────────────────────────
	LogLevel string
	LogFile  string
}

// Load reads .env (if present) and builds the Config. Missing optional values
// fall back to the same defaults the Python code uses. It never fails on a
// missing .env: startup must succeed even with no credentials at all (§5.5).
func Load(envFiles ...string) (*Config, error) {
	if len(envFiles) == 0 {
		envFiles = defaultEnvFiles()
	}
	for _, f := range envFiles {
		if _, err := os.Stat(f); err == nil {
			// Matches python-dotenv's override=False: first file to define a
			// key wins, and a real environment variable always wins.
			_ = godotenv.Load(f)
		}
	}

	c := &Config{
		Port: envInt("FLASK_PORT", 5000),
		Env:  envStr("FLASK_ENV", ""),
		CORSOrigins: []string{
			"http://localhost:5173", // Vite dev server
			"http://localhost:5174", // Vite dev server (fallback port)
			"http://localhost:5175", // Vite dev server (fallback port)
			"http://localhost:4173", // Vite preview
			"http://localhost:3000", // fallback
		},

		GEEProjectID:         envStr("GEE_PROJECT_ID", ""),
		GEEServiceAccountKey: envStr("GEE_SERVICE_ACCOUNT_KEY", "./gee-service-account.json"),
		GEEUserCredentials:   envStr("GEE_USER_CREDENTIALS", ""),
		Dataset:              "COPERNICUS/S2_SR_HARMONIZED",
		LookbackDays:         90,
		MaxCloudCoverPct:     20,
		// 3=Cloud Shadow, 8=Medium Cloud, 9=High Cloud, 10=Cirrus.
		// Order is load-bearing: the mask is built by chaining .And() in this
		// sequence, and the expression graph must match the fixture.
		SCLMaskValues: []int{3, 8, 9, 10},
		Bands: map[string]string{
			"BLUE": "B2", "GREEN": "B3", "RED": "B4",
			"NIR": "B8", "SWIR": "B11", "SCL": "SCL",
		},

		// Must sum to 1.0. These are config.py's values, NOT the README's
		// older 0.35/0.25/0.15/0.15/0.10 set — see K2.
		CVIWeights: map[string]float64{
			"NDVI": 0.70, "EVI": 0.10, "SAVI": 0.05,
			"NDMI": 0.10, "GNDVI": 0.05,
		},
		GridScaleM:     10,
		MaxGridCells:   2000,
		GridScaleStepM: 2,
		// grid_service passes sigma_factor=0.6 explicitly from both the
		// vegetation and radar paths; the function's 1.2 default is dead.
		SmoothSigmaFactor:     0.6,
		ConfidenceSceneTarget: 5,
		ConfidenceStdMax:      0.3,

		CVIThresholds: []Threshold{
			{0.5, "Healthy vegetation"},
			{0.25, "Moderate vegetation, possible stress"},
			{-1.0, "Poor vegetation, needs attention"},
		},
		NDVIThresholds: []Threshold{
			{0.6, "Dense, healthy vegetation"},
			{0.4, "Moderate vegetation"},
			{0.2, "Sparse / stressed vegetation"},
			{0.0, "Bare soil"},
			{-1.0, "Water / non-vegetated"},
		},
		EVIThresholds: []Threshold{
			{0.5, "Dense vegetation"},
			{0.3, "Moderate vegetation"},
			{0.1, "Sparse vegetation"},
			{-1.0, "Bare / non-vegetated"},
		},
		SAVIThresholds: []Threshold{
			{0.5, "High vegetation + low soil effect"},
			{0.3, "Moderate vegetation"},
			{0.1, "Low vegetation"},
			{-1.0, "Bare soil dominant"},
		},
		NDMIThresholds: []Threshold{
			{0.4, "High moisture"},
			{0.2, "Moderate moisture"},
			{0.0, "Low moisture"},
			{-1.0, "Dry / drought stress"},
		},
		NDWIThresholds: []Threshold{
			{0.3, "High water presence"},
			{0.0, "Water likely present"},
			{-1.0, "No significant water"},
		},
		GNDVIThresholds: []Threshold{
			{0.6, "Excellent chlorophyll / nutrient status"},
			{0.4, "Good chlorophyll"},
			{0.2, "Moderate"},
			{-1.0, "Low chlorophyll"},
		},
		SMIThresholds: []Threshold{
			{0.66, "Wet"},
			{0.33, "Moderate"},
			{0.0, "Dry"},
		},

		S1Dataset:        "COPERNICUS/S1_GRD",
		S1InstrumentMode: "IW",
		S1OrbitPass:      "DESCENDING",
		S1SpeckleKernel:  "circle",
		S1ResolutionM:    10,
		S1LookbackDays:   90,
		S1DateWindowDays: 6,
		S1SpeckleRadiusM: 50,
		SMIVVDryDB:       -20.0,
		SMIVVWetDB:       -8.0,
		RadarBands:       []string{"SMI", "RVI", "RATIO", "VV", "VH"},
		RadarVisBounds: map[string]VisBound{
			"SMI":   {0.0, 1.0},
			"RVI":   {0.0, 1.0},
			"RATIO": {2.0, 16.0},
			"VV":    {-22.0, -6.0},
			"VH":    {-28.0, -12.0},
		},

		// Palettes are copied verbatim from app.py. They are posted to Earth
		// Engine inside Image.visualize, WITH the leading '#' — the client does
		// not strip it (verified against a live maps request; see
		// internal/gee/eeexpr/testdata/maps_request_response.json).
		NDVIPalette: []string{
			"#ad0028", "#c5142a", "#e02d2c", "#ef4c3a", "#fe6c4a",
			"#ff8d5a", "#ffab69", "#ffc67d", "#ffe093", "#ffefab",
			"#fdfec2", "#eaf7ac", "#d5ef94", "#b9e383", "#9bd873",
			"#77ca6f", "#53bd6b", "#14aa60", "#009755", "#007e47", "#007e47",
		},
		CVIPalette: []string{"#ef4444", "#f59e0b", "#22c55e"},
		RadarMoisturePalette: []string{
			"#b30000", "#e34a33", "#fc8d59", "#fdcc8a",
			"#ffffbf", "#c2e699", "#78c679", "#31a354", "#006837",
		},
		RadarSequentialPalette: []string{
			"#f7fbff", "#c6dbef", "#6baed6", "#3182bd", "#08519c",
		},

		DatabaseURL:       envStr("DATABASE_URL", ""),
		FirebaseProjectID: envStr("FIREBASE_PROJECT_ID", ""),
		ServiceAccountKey: envStr("SERVICE_ACCOUNT_KEY", "serviceAccountKey.json"),
		JWTSecret:         envStr("JWT_SECRET_KEY", "dev-secret-change-me"),
		JWTExpiry:         7 * 24 * time.Hour,
		OllamaBaseURL:     envStr("OLLAMA_BASE_URL", ""),
		OllamaModel:       envStr("OLLAMA_MODEL", ""),
		OllamaTemperature: envFloat("OLLAMA_TEMPERATURE", 0.7),
		OllamaMaxTokens:   envInt("OLLAMA_MAX_TOKENS", 512),
		ChatbotMaxHistory: envInt("CHATBOT_MAX_HISTORY", 20),
		SMSUsername:       envStr("SMS_USERNAME", ""),
		SMSPassword:       envStr("SMS_PASSWORD", ""),
		SMSFrom:           envStr("SMS_FROM", ""),
		SMSDLTContentID:   envStr("SMS_DLT_CONTENT_ID", ""),
		SMSDLTPEID:        envStr("SMS_DLT_PE_ID", ""),
		OTPExpiry:         600 * time.Second,

		ExprCacheTTL:        30 * time.Minute,
		ExprCacheMaxEntries: 256,

		DBMinConns: 2,
		DBMaxConns: 20,

		LogLevel: envStr("LOG_LEVEL", "INFO"),
		LogFile:  "cvi_engine.log",
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// defaultEnvFiles lists the .env candidates, relative to the working directory
// AND to the repository root.
//
// Only the repo root is searched now. Before the v2.0.0-go cutover the file
// lived at legacy-python/.env; it was moved to the root when that directory was
// deleted.
//
// The root walk matters because `go test ./internal/gee/` runs with the working
// directory set to that package, and because the server binary may be started
// from anywhere. Without it, a test that needs real credentials fails with a
// confusing "GEE_PROJECT_ID is not set" rather than finding the .env two
// directories up.
func defaultEnvFiles() []string {
	rel := []string{".env"}
	out := append([]string{}, rel...)
	if root, ok := repoRoot(); ok {
		for _, r := range rel {
			out = append(out, filepath.Join(root, filepath.FromSlash(r)))
		}
	}
	return out
}

// repoRoot walks up from the working directory looking for go.mod.
func repoRoot() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

// Validate checks the invariants config.py documents but never enforces.
func (c *Config) Validate() error {
	var sum float64
	for _, w := range c.CVIWeights {
		sum += w
	}
	if diff := sum - 1.0; diff > 1e-9 || diff < -1e-9 {
		return fmt.Errorf("CVI weights must sum to 1.0, got %v", sum)
	}
	for name, ts := range map[string][]Threshold{
		"CVI": c.CVIThresholds, "NDVI": c.NDVIThresholds, "EVI": c.EVIThresholds,
		"SAVI": c.SAVIThresholds, "NDMI": c.NDMIThresholds, "NDWI": c.NDWIThresholds,
		"GNDVI": c.GNDVIThresholds, "SMI": c.SMIThresholds,
	} {
		for i := 1; i < len(ts); i++ {
			if ts[i].Min >= ts[i-1].Min {
				return fmt.Errorf("%s thresholds must be sorted descending: %v >= %v at index %d",
					name, ts[i].Min, ts[i-1].Min, i)
			}
		}
	}
	return nil
}

// Interpret maps a value to a label using a descending threshold table.
// A nil value yields noData; a value below every threshold yields noMatch.
// This mirrors stats_service.interpret_value and grid_service._interpret_cvi.
func Interpret(v *float64, table []Threshold, noData, noMatch string) string {
	if v == nil {
		return noData
	}
	for _, t := range table {
		if *v >= t.Min {
			return t.Label
		}
	}
	return noMatch
}

func envStr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return f
		}
	}
	return def
}

package config

import (
	"strings"
	"testing"
	"time"
)

// baseValid returns a Config that passes every check except the ones a test
// deliberately breaks.
func baseValid() *Config {
	return &Config{
		JWTSecret: strings.Repeat("k", minProdJWTSecretLen),
		CVIWeights: map[string]float64{
			"NDVI": 0.70, "EVI": 0.10, "SAVI": 0.05, "NDMI": 0.10, "GNDVI": 0.05,
		},
		CVIThresholds: []Threshold{{0.5, "a"}, {0.25, "b"}, {-1.0, "c"}},
	}
}

func TestValidateRejectsDevSecretInProduction(t *testing.T) {
	c := baseValid()
	c.Env = "production"
	c.JWTSecret = DevJWTSecret

	err := c.Validate()
	if err == nil {
		t.Fatal("production must refuse to start on the built-in development key")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET_KEY") {
		t.Errorf("the error should name the variable to set, got: %v", err)
	}
}

func TestValidateRejectsEmptyAndShortSecretsInProduction(t *testing.T) {
	for name, secret := range map[string]string{
		"empty":      "",
		"whitespace": "   ",
		"too short":  "short",
	} {
		c := baseValid()
		c.Env = "production"
		c.JWTSecret = secret
		if err := c.Validate(); err == nil {
			t.Errorf("%s secret should be rejected in production", name)
		}
	}
}

func TestValidateAcceptsStrongSecretInProduction(t *testing.T) {
	c := baseValid()
	c.Env = "production"
	if err := c.Validate(); err != nil {
		t.Fatalf("a %d-character secret should be accepted: %v", minProdJWTSecretLen, err)
	}
}

// The offline test suite and local dev run with no environment at all, so the
// development key must stay usable outside production — just noisily.
func TestValidateAllowsDevSecretOutsideProduction(t *testing.T) {
	for _, env := range []string{"", "development", "test"} {
		c := baseValid()
		c.Env = env
		c.JWTSecret = DevJWTSecret
		if err := c.Validate(); err != nil {
			t.Errorf("FLASK_ENV=%q should tolerate the development key, got: %v", env, err)
		}
	}
}

func TestIsProductionIsCaseAndSpaceInsensitive(t *testing.T) {
	for _, env := range []string{"production", "Production", "PRODUCTION", " production "} {
		if !(&Config{Env: env}).IsProduction() {
			t.Errorf("FLASK_ENV=%q should count as production", env)
		}
	}
	for _, env := range []string{"", "dev", "staging", "productionish"} {
		if (&Config{Env: env}).IsProduction() {
			t.Errorf("FLASK_ENV=%q should NOT count as production", env)
		}
	}
}

func TestUsingDevJWTSecret(t *testing.T) {
	if !(&Config{JWTSecret: DevJWTSecret}).UsingDevJWTSecret() {
		t.Error("the built-in key should be reported as the development key")
	}
	if (&Config{JWTSecret: "something-else"}).UsingDevJWTSecret() {
		t.Error("a real key must not be reported as the development key")
	}
}

func TestEnvListSplitsAndFallsBack(t *testing.T) {
	def := []string{"http://localhost:5173"}

	t.Setenv("TEST_CORS", "https://a.example, https://b.example ,")
	got := envList("TEST_CORS", def)
	if len(got) != 2 || got[0] != "https://a.example" || got[1] != "https://b.example" {
		t.Fatalf("envList should split, trim and drop blanks; got %#v", got)
	}

	// An all-blank value must not produce an empty allow-list, which would
	// silently break every browser client.
	t.Setenv("TEST_CORS", " , ,")
	if got := envList("TEST_CORS", def); len(got) != 1 || got[0] != def[0] {
		t.Fatalf("an all-blank value should fall back to the default; got %#v", got)
	}
}

func TestEnvBoolSpellings(t *testing.T) {
	for _, v := range []string{"1", "true", "TRUE", "yes", "on"} {
		t.Setenv("TEST_BOOL", v)
		if !envBool("TEST_BOOL", false) {
			t.Errorf("%q should parse as true", v)
		}
	}
	for _, v := range []string{"0", "false", "No", "off"} {
		t.Setenv("TEST_BOOL", v)
		if envBool("TEST_BOOL", true) {
			t.Errorf("%q should parse as false", v)
		}
	}
	// An unparseable value keeps the default rather than silently disabling.
	t.Setenv("TEST_BOOL", "maybe")
	if !envBool("TEST_BOOL", true) {
		t.Error("an unrecognised value should keep the default")
	}
}

func TestLoadDefaultsAreSane(t *testing.T) {
	c, err := Load("does-not-exist.env")
	if err != nil {
		t.Fatalf("Load with no env file should still succeed: %v", err)
	}
	if !c.RateLimitEnabled {
		t.Error("rate limiting should default to on")
	}
	if c.MaxRequestBytes <= 0 {
		t.Error("a request-body cap must have a positive default")
	}
	if c.OTPMaxAttempts <= 0 {
		t.Error("the OTP attempt budget must be positive")
	}
	if c.ChatSessionTTL <= 0 || c.ChatMaxSessions <= 0 {
		t.Error("chat retention bounds must be positive")
	}
	if c.AnalyzeMaxInFlight <= 0 {
		t.Error("the analysis concurrency gate must be positive")
	}
	// Generous enough that the 74-case contract replay cannot trip it.
	if c.RateLimitPerMin < 100 {
		t.Errorf("the default rate limit (%d/min) is too tight for the contract suite",
			c.RateLimitPerMin)
	}
	_ = time.Second
}

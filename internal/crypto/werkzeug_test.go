package crypto

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVerifyAgainstPythonFixtures is the acceptance test §8.2 calls for: every
// hash here was produced by the real Werkzeug in the project's venv. If the
// dkLen or the salt encoding were wrong, this fails immediately — which is the
// entire point of generating the fixtures rather than reasoning about the format.
func TestVerifyAgainstPythonFixtures(t *testing.T) {
	type fixture struct {
		WerkzeugVersion     string `json:"werkzeug_version"`
		DefaultMethodPrefix string `json:"default_method_prefix"`
		Cases               []struct {
			Password string `json:"password"`
			Method   string `json:"method"`
			Hash     string `json:"hash"`
		} `json:"cases"`
	}

	raw, err := os.ReadFile(filepath.Join("testdata", "werkzeug_hashes.json"))
	if err != nil {
		t.Skipf("fixtures not present: %v", err)
	}
	var f fixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("decode fixtures: %v", err)
	}
	if len(f.Cases) == 0 {
		t.Fatal("no fixture cases")
	}
	t.Logf("werkzeug %s, default method %s, %d cases",
		f.WerkzeugVersion, f.DefaultMethodPrefix, len(f.Cases))

	// The production rows on this deployment use scrypt:32768:8:1, so that
	// path in particular must work.
	if !strings.HasPrefix(f.DefaultMethodPrefix, "scrypt:") {
		t.Errorf("expected a scrypt default, got %q", f.DefaultMethodPrefix)
	}

	for _, c := range f.Cases {
		t.Run(c.Method+"/"+truncate(c.Password), func(t *testing.T) {
			ok, err := VerifyPassword(c.Hash, c.Password)
			if err != nil {
				t.Fatalf("VerifyPassword: %v", err)
			}
			if !ok {
				t.Errorf("correct password rejected for method %s", c.Method)
			}

			// And the negative case: a wrong password must not verify.
			bad, err := VerifyPassword(c.Hash, c.Password+"x")
			if err != nil {
				t.Fatalf("VerifyPassword (wrong): %v", err)
			}
			if bad {
				t.Error("wrong password accepted")
			}
		})
	}
}

func TestGenerateRoundTrip(t *testing.T) {
	for _, pw := range []string{"password123", "hunter2", "aA1!£€ 😀", strings.Repeat("x", 72), ""} {
		h, err := GeneratePassword(pw)
		if err != nil {
			t.Fatalf("GeneratePassword(%q): %v", truncate(pw), err)
		}
		if !strings.HasPrefix(h, "scrypt:32768:8:1$") {
			t.Errorf("hash %q does not use the Werkzeug default method", h[:min(40, len(h))])
		}
		parts := strings.Split(h, "$")
		if len(parts) != 3 {
			t.Fatalf("hash has %d parts, want 3", len(parts))
		}
		if len(parts[1]) != saltLength {
			t.Errorf("salt length %d, want %d", len(parts[1]), saltLength)
		}
		// scrypt dkLen 64 → 128 hex characters.
		if len(parts[2]) != scryptDKLen*2 {
			t.Errorf("digest length %d, want %d", len(parts[2]), scryptDKLen*2)
		}

		ok, err := VerifyPassword(h, pw)
		if err != nil || !ok {
			t.Errorf("round trip failed for %q: ok=%v err=%v", truncate(pw), ok, err)
		}
		bad, _ := VerifyPassword(h, pw+"x")
		if bad {
			t.Errorf("wrong password accepted for %q", truncate(pw))
		}
	}
}

func TestSaltIsRandom(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 32; i++ {
		h, err := GeneratePassword("same-password")
		if err != nil {
			t.Fatal(err)
		}
		salt := strings.Split(h, "$")[1]
		if seen[salt] {
			t.Fatalf("salt %q repeated within 32 generations", salt)
		}
		seen[salt] = true
	}
}

func TestMalformedHashesAreErrors(t *testing.T) {
	// A corrupt row must be distinguishable from a wrong password, so these
	// return an error rather than a silent false.
	for _, bad := range []string{
		"",
		"nodollarsigns",
		"only$two",
		"$$",
		"md5$salt$deadbeef",
		"scrypt:notanumber:8:1$salt$deadbeef",
		"scrypt:32768:8$salt$deadbeef",
	} {
		if _, err := VerifyPassword(bad, "x"); err == nil {
			t.Errorf("VerifyPassword(%q) returned no error", bad)
		}
	}
}

// TestPBKDF2Path covers the legacy method explicitly: §8.3 says be liberal in
// what you accept, and an older row may still carry a pbkdf2 hash.
func TestPBKDF2Path(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "werkzeug_hashes.json"))
	if err != nil {
		t.Skipf("fixtures not present: %v", err)
	}
	var f struct {
		Cases []struct {
			Password string `json:"password"`
			Method   string `json:"method"`
			Hash     string `json:"hash"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, c := range f.Cases {
		if !strings.HasPrefix(c.Hash, "pbkdf2:") {
			continue
		}
		found++
		ok, err := VerifyPassword(c.Hash, c.Password)
		if err != nil || !ok {
			t.Errorf("pbkdf2 verify failed: ok=%v err=%v", ok, err)
		}
	}
	if found == 0 {
		t.Error("no pbkdf2 fixtures present — the legacy path is untested")
	}
}

func truncate(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12] + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

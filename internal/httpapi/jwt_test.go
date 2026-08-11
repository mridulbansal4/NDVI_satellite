package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
)

func testSecret(t *testing.T) []byte {
	t.Helper()
	cfg, err := config.Load("\x00nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	return []byte(cfg.JWTSecret)
}

func testToken(t *testing.T) string {
	t.Helper()
	return mint(t, testSecret(t), time.Now().Add(time.Hour))
}

func mint(t *testing.T, secret []byte, exp time.Time) string {
	t.Helper()
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "00000000-0000-0000-0000-0000000000c1",
		"type":  "access",
		"fresh": false,
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"exp":   exp.Unix(),
		"jti":   "test",
	})
	s, err := tok.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TestJWTMiddlewareBranches pins the four failure modes against the responses
// captured from the running Flask app (testdata/golden/jwt_*.json).
//
// PRD §8.3's table is WRONG about the malformed-header row: it claims 422 with
// "Bad Authorization header. Expected …", while flask-jwt-extended 4.7.4
// actually answers 401 with the wording below. Getting 401-vs-422 backwards
// here breaks the frontend's axios interceptor and therefore the logout flow.
func TestJWTMiddlewareBranches(t *testing.T) {
	h, _, _ := newTestServer(t)
	secret := testSecret(t)

	cases := []struct {
		name       string
		header     string
		wantStatus int
		wantMsg    string
	}{
		{
			"no header", "",
			http.StatusUnauthorized, "Missing Authorization Header",
		},
		{
			"not a Bearer header", "some-token-without-a-scheme",
			http.StatusUnauthorized,
			"Missing 'Bearer' type in 'Authorization' header. Expected 'Authorization: Bearer <JWT>'",
		},
		{
			"wrong signing key", "Bearer " + mint(t, []byte("wrong-secret"), time.Now().Add(time.Hour)),
			http.StatusUnprocessableEntity, "Signature verification failed",
		},
		{
			"expired", "Bearer " + mint(t, secret, time.Now().Add(-time.Minute)),
			http.StatusUnauthorized, "Token has expired",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/dashboard", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status %d, want %d", w.Code, tc.wantStatus)
			}
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body["msg"] != tc.wantMsg {
				t.Errorf("msg = %q, want %q", body["msg"], tc.wantMsg)
			}
			if _, hasError := body["error"]; hasError {
				t.Error(`auth failures use the {"msg":…} envelope, not {"error":…}`)
			}
		})
	}
}

// TestJWTAcceptsLiberally covers §8.3's "be liberal in what you accept": a
// token carrying only sub/exp — no type, fresh or jti — must still work, so
// tokens already in users' localStorage keep working after the cutover.
func TestJWTAcceptsLiberally(t *testing.T) {
	h, _, _ := newTestServer(t)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "00000000-0000-0000-0000-0000000000c1",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	s, err := tok.SignedString(testSecret(t))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+s)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code == http.StatusUnauthorized || w.Code == http.StatusUnprocessableEntity {
		t.Errorf("minimal token rejected with %d: %s", w.Code, w.Body.String())
	}
}

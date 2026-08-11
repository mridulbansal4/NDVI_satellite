// Package jwtutil issues Flask-JWT-Extended-compatible access tokens.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §8.3.
//
// Verification lives in internal/httpapi/middleware/jwt.go and is deliberately
// LIBERAL — any HS256 token with a non-empty string `sub` that has not expired
// is accepted, so tokens already sitting in users' localStorage under
// `agri_token` keep working. Issuance is deliberately CONSERVATIVE: it emits
// the full claim set flask-jwt-extended produces, so a rollback to the Python
// backend keeps working too.
package jwtutil

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Issuer mints access tokens.
type Issuer struct {
	Secret []byte
	Expiry time.Duration
	// Now is injectable for tests.
	Now func() time.Time
}

// NewIssuer returns an Issuer for the configured secret and lifetime.
func NewIssuer(secret string, expiry time.Duration) *Issuer {
	return &Issuer{Secret: []byte(secret), Expiry: expiry}
}

func (i *Issuer) now() time.Time {
	if i.Now != nil {
		return i.Now()
	}
	return time.Now()
}

// Issue mints an access token for a farmer id.
//
// The claim set was read off a token minted by the running Flask app
// (testdata/golden/e11_signup_success.json), NOT transcribed from the PRD —
// §8.3's list omits `csrf`, which flask-jwt-extended does emit. `sub` must be a
// STRING; flask-jwt-extended rejects a non-string identity.
func (i *Issuer) Issue(farmerID string) (string, error) {
	now := i.now()
	iat := now.Unix()

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   farmerID,
		"type":  "access",
		"fresh": false,
		"iat":   iat,
		"nbf":   iat, // flask-jwt-extended sets nbf == iat
		"exp":   now.Add(i.Expiry).Unix(),
		"jti":   uuid.NewString(),
		// Emitted because the Python does, even though nothing consumes it:
		// CSRF protection is only active for cookie-based tokens, and this
		// deployment sends the token in the Authorization header.
		"csrf": uuid.NewString(),
	})
	return tok.SignedString(i.Secret)
}

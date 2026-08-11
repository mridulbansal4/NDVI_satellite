// Package middleware holds the Gin middleware that replaces Flask's decorators.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §8.3.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// FarmerIDKey is the Gin-context key holding the verified identity. It is the
// direct analogue of Flask's g.farmer_id.
const FarmerIDKey = "farmer_id"

// JWT verifies a Flask-JWT-Extended-compatible HS256 token.
//
// The failure statuses and bodies below were captured from the running Flask
// app (testdata/golden/jwt_*.json), NOT transcribed from PRD §8.3 — the PRD's
// table is wrong about the malformed-header case, claiming 422 with different
// wording where flask-jwt-extended 4.7.4 actually answers 401.
//
//	no Authorization header  → 401 {"msg":"Missing Authorization Header"}
//	header not "Bearer <t>"  → 401 {"msg":"Missing 'Bearer' type in 'Authorization' header. Expected 'Authorization: Bearer <JWT>'"}
//	signature invalid        → 422 {"msg":"Signature verification failed"}
//	expired                  → 401 {"msg":"Token has expired"}
//
// Verification is deliberately liberal (§8.3): any token that verifies under
// HS256, is unexpired and carries a non-empty string `sub` is accepted. The
// type/fresh/jti claims are emitted on the issue path but never required here,
// so tokens already sitting in users' localStorage keep working.
func JWT(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if strings.TrimSpace(header) == "" {
			abort(c, http.StatusUnauthorized, "Missing Authorization Header")
			return
		}

		parts := strings.Fields(header)
		if len(parts) != 2 || parts[0] != "Bearer" {
			abort(c, http.StatusUnauthorized,
				"Missing 'Bearer' type in 'Authorization' header. Expected 'Authorization: Bearer <JWT>'")
			return
		}

		var claims jwt.MapClaims
		_, err := jwt.ParseWithClaims(parts[1], &claims,
			func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return secret, nil
			},
			jwt.WithValidMethods([]string{"HS256"}),
		)
		if err != nil {
			// Expiry is reported as 401; every other verification failure as
			// 422, matching flask-jwt-extended's split.
			if strings.Contains(err.Error(), jwt.ErrTokenExpired.Error()) {
				abort(c, http.StatusUnauthorized, "Token has expired")
				return
			}
			abort(c, http.StatusUnprocessableEntity, "Signature verification failed")
			return
		}

		sub, _ := claims["sub"].(string)
		if sub == "" {
			abort(c, http.StatusUnprocessableEntity,
				`Missing claim: sub`)
			return
		}

		c.Set(FarmerIDKey, sub)
		c.Next()
	}
}

// FarmerID returns the verified identity set by JWT.
func FarmerID(c *gin.Context) string {
	v, _ := c.Get(FarmerIDKey)
	s, _ := v.(string)
	return s
}

func abort(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"msg": msg})
}

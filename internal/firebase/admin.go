// Package firebase verifies Firebase ID tokens for the phone-auth flow.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §8.4, §13.7.
//
// This is a THIRD, independent auth system alongside password auth and the SMS
// OTP path. §8.1 says not to unify them, and this one issues no app JWT — it
// only confirms the Firebase token is genuine.
package firebase

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// ErrInvalidToken carries the exact message verify_jwt_token raises, which the
// handler surfaces as 401 (§8.4).
var ErrInvalidToken = errors.New("Invalid or expired authentication token")

// ErrNotConfigured means no Firebase credentials were found.
var ErrNotConfigured = errors.New("Firebase not initialized")

// Admin verifies ID tokens.
//
// Initialisation is cached with sync.Once including the failure, so an
// unconfigured deployment does not pay a credential-discovery timeout on every
// request (§13.7).
type Admin struct {
	once sync.Once
	auth *auth.Client
	err  error

	keyPath   string
	projectID string
	log       *slog.Logger
}

// NewAdmin returns a lazily-initialised verifier.
func NewAdmin(keyPath, projectID string, log *slog.Logger) *Admin {
	return &Admin{keyPath: keyPath, projectID: projectID, log: log}
}

// clientContext is the context the Firebase SDK uses for the LIFETIME of the
// client, which is deliberately NOT the caller's.
//
// firebase.NewApp and app.Auth capture the context they are given and reuse it
// for every subsequent OAuth token refresh. Because construction happens inside
// sync.Once, the FIRST caller's context is the one that sticks — and that first
// caller is the startup probe in cmd/server, whose 30-second context is
// cancelled the moment the probe goroutine returns. The cached auth.Client
// would then hold a dead context and every refresh after the initial token
// expired (~1h) would fail with "context canceled", 401-ing every user until
// the process restarted.
//
// This is the same bug, and the same fix, as internal/gee/session.go's
// clientContext — see the commentary there.
func clientContext() context.Context { return context.Background() }

func (a *Admin) client(ctx context.Context) (*auth.Client, error) {
	_ = ctx // see clientContext: the refresh loop must outlive the caller
	a.once.Do(func() {
		ctx := clientContext()
		conf := &firebase.Config{ProjectID: a.projectID}
		var app *firebase.App
		var err error

		switch {
		case a.keyPath != "" && fileExists(a.keyPath):
			// staticcheck flags this as deprecated because it does not validate a
			// credential file that came from an untrusted source. Here the path
			// is operator-supplied configuration on the server's own filesystem.
			//lint:ignore SA1019 operator-supplied key path, not untrusted input
			app, err = firebase.NewApp(ctx, conf, option.WithCredentialsFile(a.keyPath))
		case os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "":
			app, err = firebase.NewApp(ctx, conf)
		default:
			a.log.Warn("serviceAccountKey.json not found — Firebase auth disabled")
			a.err = ErrNotConfigured
			return
		}
		if err != nil {
			a.log.Error("Failed to initialize Firebase Admin: " + err.Error())
			a.err = err
			return
		}
		a.auth, a.err = app.Auth(ctx)
		if a.err == nil {
			a.log.Info("Firebase Admin initialized successfully.")
		}
	})
	return a.auth, a.err
}

// Available reports whether Firebase is usable — this is /health's
// firebase_ready flag. false is the normal dev state (§13.7).
func (a *Admin) Available(ctx context.Context) bool {
	c, err := a.client(ctx)
	return err == nil && c != nil
}

// User is the subset of the decoded token the response carries.
//
// PhoneNumber is a pointer because Python's .get() yields None when the claim
// is absent, and the contract requires JSON null rather than "" (§8.4).
type User struct {
	UID         string  `json:"uid"`
	PhoneNumber *string `json:"phone_number"`
}

// VerifyIDToken checks a Firebase ID token and extracts uid + phone_number.
//
// Any verification failure collapses to ErrInvalidToken, matching the Python,
// which catches every exception and re-raises one ValueError.
func (a *Admin) VerifyIDToken(ctx context.Context, idToken string) (*User, error) {
	client, err := a.client(ctx)
	if err != nil {
		return nil, err
	}

	tok, err := client.VerifyIDToken(ctx, idToken)
	if err != nil {
		a.log.Error("Failed to verify JWT token: " + err.Error())
		return nil, ErrInvalidToken
	}

	u := &User{UID: tok.UID}
	if raw, ok := tok.Claims["phone_number"]; ok {
		if s, isStr := raw.(string); isStr {
			u.PhoneNumber = &s
		}
	}
	a.log.Info("Successfully verified user token for UID: " + u.UID)
	return u, nil
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

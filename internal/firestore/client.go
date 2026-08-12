// Package firestore mirrors backend/firestore/, providing the onboarding
// session and farm-alert documents.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §7.10, §13.7.
//
// Firestore is OPTIONAL. serviceAccountKey.json is gitignored and absent in
// dev, so an unconfigured Firestore is the normal local state and must never
// fail a request: every write here is best-effort and logs at WARN.
package firestore

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"

	fs "cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
)

// ErrNotConfigured means no credentials were found.
var ErrNotConfigured = errors.New(
	"Firestore disabled: no serviceAccountKey.json and GOOGLE_APPLICATION_CREDENTIALS not set")

// Client wraps the Firestore client with once-only initialisation.
//
// §13.7: the init failure is cached so requests do not each pay a repeated
// credential-discovery timeout — the Python added _init_failed for exactly this
// reason, and blind Application Default Credentials discovery in dev costs
// multiple seconds per call against the metadata server.
type Client struct {
	once sync.Once
	db   *fs.Client
	err  error

	cfg *config.Config
	log *slog.Logger
}

// New returns a lazily-initialised client.
func New(cfg *config.Config, log *slog.Logger) *Client {
	return &Client{cfg: cfg, log: log}
}

// DB returns the Firestore client, initialising on first use.
//
// Credential resolution order matches firestore/client.py:
//  1. serviceAccountKey.json in the repo root
//  2. GOOGLE_APPLICATION_CREDENTIALS → Application Default Credentials
//  3. neither → disabled
// clientContext is the context the Firestore SDK uses for the LIFETIME of the
// client, which is deliberately NOT the caller's.
//
// firebase.NewApp and app.Firestore capture the context they are given and
// reuse it for every later OAuth token refresh. Construction happens inside
// sync.Once, so the FIRST caller's context is the one that sticks — and here
// that is a request context, cancelled as soon as the response is written. The
// cached client would hold a dead context and every write after the initial
// token expired would fail with "context canceled".
//
// Same bug and same fix as internal/gee/session.go's clientContext.
func clientContext() context.Context { return context.Background() }

func (c *Client) DB(ctx context.Context) (*fs.Client, error) {
	_ = ctx // see clientContext: the refresh loop must outlive the caller
	c.once.Do(func() {
		ctx := clientContext()
		keyPath := c.cfg.ServiceAccountKey
		conf := &firebase.Config{ProjectID: c.cfg.FirebaseProjectID}

		var app *firebase.App
		var err error
		switch {
		case keyPath != "" && fileExists(keyPath):
			c.log.Info("[Firestore] Using serviceAccountKey.json")
			// staticcheck flags this as deprecated because it does not validate a
			// credential file that came from an untrusted source. Here the path
			// is operator-supplied configuration on the server's own filesystem.
			//lint:ignore SA1019 operator-supplied key path, not untrusted input
			app, err = firebase.NewApp(ctx, conf, option.WithCredentialsFile(keyPath))
		case os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "":
			c.log.Info("[Firestore] Using Application Default Credentials")
			app, err = firebase.NewApp(ctx, conf)
		default:
			c.err = ErrNotConfigured
			return
		}
		if err != nil {
			c.err = err
			return
		}
		c.db, c.err = app.Firestore(ctx)
		if c.err == nil {
			c.log.Info("[Firestore] Client initialized.")
		}
	})
	return c.db, c.err
}

// Available reports whether Firestore is usable, without forcing init.
func (c *Client) Available(ctx context.Context) bool {
	db, err := c.DB(ctx)
	return err == nil && db != nil
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

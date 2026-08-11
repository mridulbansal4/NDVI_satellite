// Package gee owns authentication to and communication with the Earth Engine
// REST API. It is the only package that performs network I/O against Google.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §5.5, §11.2.
package gee

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
)

// Earth Engine OAuth scopes. cloud-platform is required in addition to
// earthengine because compute runs against a Cloud project.
var scopes = []string{
	"https://www.googleapis.com/auth/earthengine",
	"https://www.googleapis.com/auth/cloud-platform",
}

// installedAppCredentials resolves the OAuth client the `earthengine
// authenticate` flow uses, for the developer fallback path only.
//
// These values are NOT committed. They ship inside the earthengine-api package
// as ee.oauth.CLIENT_ID / CLIENT_SECRET (an installed-application client, which
// by OAuth design cannot keep a secret), and are read at runtime from either:
//
//  1. GEE_OAUTH_CLIENT_ID / GEE_OAUTH_CLIENT_SECRET, or
//  2. the installed earthengine-api package's oauth.py, if a Python
//     environment with it is present.
//
// If neither is available the fallback is simply unavailable and the operator
// is told to provision a service account, which is the production path anyway.
func installedAppCredentials() (clientID, clientSecret string, err error) {
	clientID = os.Getenv("GEE_OAUTH_CLIENT_ID")
	clientSecret = os.Getenv("GEE_OAUTH_CLIENT_SECRET")
	if clientID != "" && clientSecret != "" {
		return clientID, clientSecret, nil
	}

	id, secret, perr := readEEOAuthConstants()
	if perr != nil {
		return "", "", fmt.Errorf(
			"the developer credential fallback needs the earthengine-api OAuth "+
				"client. Set GEE_OAUTH_CLIENT_ID and GEE_OAUTH_CLIENT_SECRET, or "+
				"provision a service account and set GEE_SERVICE_ACCOUNT_KEY "+
				"(the production path): %w", perr)
	}
	return id, secret, nil
}

// eeOAuthSearchPaths lists the places an installed earthengine-api may live,
// resolved against both the working directory and the repository root.
//
// The root walk matters because `go test ./internal/gee/` runs with the working
// directory set to that package.
func eeOAuthSearchPaths() []string {
	rel := []string{
		filepath.Join("legacy-python", "venv", "Lib", "site-packages", "ee", "oauth.py"),
		filepath.Join("legacy-python", "venv", "lib", "site-packages", "ee", "oauth.py"),
		filepath.Join("backend", "venv", "Lib", "site-packages", "ee", "oauth.py"),
	}
	out := append([]string{}, rel...)
	if root, ok := repoRoot(); ok {
		for _, r := range rel {
			out = append(out, filepath.Join(root, r))
		}
	}
	return out
}

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

// readEEOAuthConstants scrapes CLIENT_ID / CLIENT_SECRET out of the installed
// ee/oauth.py. Fragile by nature, which is why it is the last resort and why
// its failure is non-fatal.
func readEEOAuthConstants() (clientID, clientSecret string, err error) {
	paths := eeOAuthSearchPaths()
	if p := os.Getenv("EE_OAUTH_PY"); p != "" {
		paths = append([]string{p}, paths...)
	}
	for _, p := range paths {
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			continue
		}
		id := matchPyConst(string(raw), "CLIENT_ID")
		secret := matchPyConst(string(raw), "CLIENT_SECRET")
		if id != "" && secret != "" {
			return id, secret, nil
		}
	}
	return "", "", fmt.Errorf("could not locate ee/oauth.py in %v", paths)
}

// matchPyConst extracts a module-level string constant from Python source.
//
// It must cope with implicit string concatenation across lines, because
// ee/oauth.py writes:
//
//	CLIENT_ID = ('517222506229-vsmmajv00ul0bs7p89v5m89qs8eb9359.'
//	             'apps.googleusercontent.com')
//
// so a single-literal regex silently returns only the first fragment — or, with
// the leading parenthesis, nothing at all.
func matchPyConst(src, name string) string {
	assign := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(name) + `\s*=\s*`)
	loc := assign.FindStringIndex(src)
	if loc == nil {
		return ""
	}
	rest := src[loc[1]:]

	// Bound the statement: either to the matching ')' or to the end of the line.
	end := len(rest)
	if strings.HasPrefix(strings.TrimLeft(rest, " \t"), "(") {
		depth := 0
		for i, r := range rest {
			switch r {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					end = i
					goto bounded
				}
			}
		}
	} else if i := strings.IndexByte(rest, '\n'); i >= 0 {
		end = i
	}
bounded:
	stmt := rest[:end]

	// Concatenate every string literal in the statement.
	lit := regexp.MustCompile(`'([^']*)'|"([^"]*)"`)
	var sb strings.Builder
	for _, m := range lit.FindAllStringSubmatch(stmt, -1) {
		if m[1] != "" {
			sb.WriteString(m[1])
			continue
		}
		sb.WriteString(m[2])
	}
	return sb.String()
}

// storedCredentials is the on-disk shape of ~/.config/earthengine/credentials.
//
// VERIFIED against the real file (PRD §5.5 marks this as must-verify, and its
// example is wrong): the file contains ONLY refresh_token, redirect_uri and
// scopes. There is no client_id or client_secret — those come from the library
// constants above.
type storedCredentials struct {
	RefreshToken string   `json:"refresh_token"`
	RedirectURI  string   `json:"redirect_uri"`
	Scopes       []string `json:"scopes"`
}

// Session holds an authenticated HTTP client for the Earth Engine REST API.
type Session struct {
	HTTP      *http.Client
	ProjectID string
	Source    string // "service-account" | "user-credentials", for logging
}

// NewSession resolves credentials and returns an authenticated session.
//
// Resolution order:
//  1. GEE_SERVICE_ACCOUNT_KEY — the production path (objective O7: startup must
//     not depend on an interactive browser OAuth flow).
//  2. The stored `earthengine authenticate` refresh token — a developer
//     convenience so someone who has already authenticated can run the Go
//     server without provisioning a service account.
//
// An error here is NOT fatal to the process: the caller logs it, leaves
// gee_ready false and still serves /health, /auth/* and /dashboard, which is
// current behaviour that the frontend relies on (§5.5).
func NewSession(ctx context.Context, cfg *config.Config) (*Session, error) {
	if cfg.GEEProjectID == "" {
		return nil, fmt.Errorf(
			"GEE_PROJECT_ID is not set. Create a .env file with:\n" +
				"    GEE_PROJECT_ID=your-cloud-project-id\n" +
				"Find your project ID at https://console.cloud.google.com")
	}

	if ts, err := serviceAccountTokenSource(ctx, cfg.GEEServiceAccountKey); err == nil {
		return &Session{
			HTTP:      oauth2.NewClient(ctx, ts),
			ProjectID: cfg.GEEProjectID,
			Source:    "service-account",
		}, nil
	} else if !os.IsNotExist(err) && !strings.Contains(err.Error(), "no such file") {
		// A key that exists but is malformed is worth surfacing rather than
		// silently falling through to the developer path.
		return nil, fmt.Errorf("service-account key %s: %w", cfg.GEEServiceAccountKey, err)
	}

	ts, err := userCredentialsTokenSource(ctx, cfg.GEEUserCredentials)
	if err != nil {
		return nil, fmt.Errorf(
			"no Earth Engine credentials: set GEE_SERVICE_ACCOUNT_KEY to a "+
				"service-account JSON key registered for Earth Engine, or run "+
				"`earthengine authenticate` once: %w", err)
	}
	return &Session{
		HTTP:      oauth2.NewClient(ctx, ts),
		ProjectID: cfg.GEEProjectID,
		Source:    "user-credentials",
	}, nil
}

func serviceAccountTokenSource(ctx context.Context, path string) (oauth2.TokenSource, error) {
	if path == "" {
		return nil, os.ErrNotExist
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	creds, err := google.CredentialsFromJSON(ctx, data, scopes...)
	if err != nil {
		return nil, err
	}
	return creds.TokenSource, nil
}

// userCredentialsTokenSource reuses the refresh token written by
// `earthengine authenticate`. Developer convenience only — production must use
// a service account.
func userCredentialsTokenSource(ctx context.Context, override string) (oauth2.TokenSource, error) {
	path := override
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".config", "earthengine", "credentials")
	}
	path = expandHome(path)

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c storedCredentials
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if c.RefreshToken == "" {
		return nil, fmt.Errorf("%s contains no refresh_token", path)
	}

	clientID, clientSecret, err := installedAppCredentials()
	if err != nil {
		return nil, err
	}
	oauthCfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       scopes,
	}
	return oauthCfg.TokenSource(ctx, &oauth2.Token{RefreshToken: c.RefreshToken}), nil
}

func expandHome(p string) string {
	if !strings.HasPrefix(p, "~") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, strings.TrimPrefix(p, "~"))
}

// NewClient returns a REST client bound to this session, with the quota
// project set so user credentials are attributed correctly.
func (s *Session) NewClient() *Client {
	return &Client{
		HTTP:         s.HTTP,
		ProjectID:    s.ProjectID,
		Base:         DefaultBase,
		QuotaProject: s.ProjectID,
	}
}

// Probe is the connectivity check. The Python code runs ee.Number(1).getInfo();
// the equivalent here is a value:compute POST of the constant 1.
//
// It runs ONCE at startup in a goroutine, not per request — the Python
// @app.before_request hook is a workaround for Flask's lifecycle and is
// deliberately not reproduced (§5.5).
func (s *Session) Probe(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	c := s.NewClient()
	var out float64
	err := c.ComputeValue(ctx, mustConstantOne(), &out)
	if err != nil {
		return err
	}
	if out != 1 {
		return fmt.Errorf("connectivity probe returned %v, want 1", out)
	}
	return nil
}

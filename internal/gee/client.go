package gee

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/SanTiwari07/NDVI_satellite/internal/gee/eeexpr"
)

// DefaultBase is the Earth Engine REST root.
const DefaultBase = "https://earthengine.googleapis.com/v1"

// Client performs the four REST calls the backend needs (§5.2):
//
//	value:compute          ← .getInfo() on a Number / Dictionary / List
//	table:computeFeatures  ← .getInfo() on a FeatureCollection
//	maps                   ← .getMapId(visParams)
type Client struct {
	HTTP      *http.Client
	ProjectID string
	Base      string

	// QuotaProject sets X-Goog-User-Project. It is REQUIRED when authenticating
	// with user credentials (the `earthengine authenticate` fallback): without
	// it Google attributes the call to the OAuth client's own project rather
	// than to GEE_PROJECT_ID, and Earth Engine answers
	//
	//   403 "Not signed up for Earth Engine or project is not registered"
	//
	// even though the user and the project are both perfectly registered. The
	// Python client sets this header, which is why the Flask backend works with
	// exactly the same credentials. Harmless with a service account.
	QuotaProject string
}

func (c *Client) base() string {
	if c.Base == "" {
		return DefaultBase
	}
	return c.Base
}

// APIError is a non-retryable Earth Engine error carrying the server's message.
//
// The full body is preserved because it names the offending function or
// argument, which is exactly what is needed to fix a malformed graph (§13.4).
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("earth engine returned %d: %s", e.Status, e.Body)
}

// Retryable reports whether the status warrants another attempt.
//
// ONLY 429 and 5xx. A 400 means the expression graph is malformed; retrying it
// wastes 3× the latency before surfacing the real bug (§13.4).
func Retryable(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

// ComputeValue evaluates an expression to a scalar, list or dictionary.
func (c *Client) ComputeValue(ctx context.Context, expr *eeexpr.Expression, out any) error {
	body, err := json.Marshal(map[string]any{"expression": expr})
	if err != nil {
		return err
	}
	var resp struct {
		Result json.RawMessage `json:"result"`
	}
	path := fmt.Sprintf("/projects/%s/value:compute", c.ProjectID)
	if err := c.post(ctx, path, body, &resp); err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(resp.Result, out)
}

// ComputeValueNode compiles and evaluates in one step.
func (c *Client) ComputeValueNode(ctx context.Context, n eeexpr.Node, out any) error {
	expr, err := eeexpr.Compile(n)
	if err != nil {
		return err
	}
	return c.ComputeValue(ctx, expr, out)
}

// FeatureCollection is the GeoJSON shape table:computeFeatures returns.
type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

// Feature keeps geometry as raw JSON so the cell polygon round-trips
// byte-identically into the response.
type Feature struct {
	Type       string          `json:"type"`
	Geometry   json.RawMessage `json:"geometry"`
	Properties map[string]any  `json:"properties"`
}

// ComputeFeatures materialises a FeatureCollection, following pagination.
//
// §13.6: a 2000-cell grid may exceed one page. If nextPageToken is ignored,
// large fields silently return partial grids — a bug the Python client hides
// because ee's getInfo handles paging internally. maxFeatures is a runaway
// guard; exceeding it is an error rather than a silent truncation, because a
// silently short grid reads as "covered everything" when it did not.
func (c *Client) ComputeFeatures(ctx context.Context, expr *eeexpr.Expression, maxFeatures int) (*FeatureCollection, error) {
	out := &FeatureCollection{Type: "FeatureCollection"}
	pageToken := ""
	path := fmt.Sprintf("/projects/%s/table:computeFeatures", c.ProjectID)

	for page := 0; ; page++ {
		payload := map[string]any{"expression": expr}
		if pageToken != "" {
			payload["pageToken"] = pageToken
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Type          string          `json:"type"`
			Features      []Feature       `json:"features"`
			NextPageToken string          `json:"nextPageToken"`
			Error         json.RawMessage `json:"error"`
		}
		if err := c.post(ctx, path, body, &resp); err != nil {
			return nil, err
		}
		out.Features = append(out.Features, resp.Features...)

		if maxFeatures > 0 && len(out.Features) > maxFeatures {
			return nil, fmt.Errorf(
				"computeFeatures returned more than %d features (page %d) — "+
					"refusing to truncate silently", maxFeatures, page)
		}
		if resp.NextPageToken == "" {
			return out, nil
		}
		pageToken = resp.NextPageToken
	}
}

// VisParams are the visualisation settings for a tile layer.
type VisParams struct {
	Min     float64
	Max     float64
	Palette []string
}

// CreateMap creates a tile layer and returns the URL template.
//
// VERIFIED against a live call (§5.2, §5.6 — the PRD's description is wrong).
// The request body is:
//
//	{"expression": <graph>, "fileFormat": "AUTO_JPEG_PNG", "bandIds": []}
//
// There is NO visualizationOptions field. The min/max/palette are folded into
// the expression itself via Image.visualize, and palette entries keep their
// leading '#'. The response's `name` is the map resource, and tiles are served
// from {base}/{name}/tiles/{z}/{x}/{y} unauthenticated — the map id is itself
// the capability token, which is why the browser can fetch tiles directly.
func (c *Client) CreateMap(ctx context.Context, img eeexpr.Image, vis VisParams) (string, error) {
	visualised := img.Visualize(vis.Min, vis.Max, vis.Palette)
	expr, err := eeexpr.Compile(visualised.N)
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(map[string]any{
		"expression": expr,
		"fileFormat": "AUTO_JPEG_PNG",
		"bandIds":    []string{},
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		Name string `json:"name"`
	}
	path := fmt.Sprintf("/projects/%s/maps?fields=name", c.ProjectID)
	if err := c.post(ctx, path, body, &resp); err != nil {
		return "", err
	}
	if resp.Name == "" {
		return "", errors.New("maps response contained no resource name")
	}
	return fmt.Sprintf("%s/%s/tiles/{z}/{x}/{y}", c.base(), resp.Name), nil
}

// retry policy (§13.4): 3 attempts, base 1s, factor 2, ±20% jitter, cap 8s.
const (
	maxAttempts  = 3
	retryBase    = time.Second
	retryFactor  = 2
	retryCap     = 8 * time.Second
	jitterFactor = 0.2
)

// post issues the request, applying the retry policy and classifying errors.
func (c *Client) post(ctx context.Context, path string, body []byte, out any) error {
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base()+path,
			bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		if c.QuotaProject != "" {
			req.Header.Set("X-Goog-User-Project", c.QuotaProject)
		}

		resp, err := c.HTTP.Do(req)
		if err != nil {
			// A cancelled request context must abort immediately rather than
			// burning the remaining attempts (§13.3, §13.4).
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lastErr = err
			continue
		}

		raw, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		if resp.StatusCode == http.StatusOK {
			if out == nil {
				return nil
			}
			return json.Unmarshal(raw, out)
		}

		apiErr := &APIError{Status: resp.StatusCode, Body: string(raw)}
		if !Retryable(resp.StatusCode) {
			return apiErr
		}
		lastErr = apiErr
	}

	return fmt.Errorf("earth engine request failed after %d attempts: %w", maxAttempts, lastErr)
}

func backoff(attempt int) time.Duration {
	d := retryBase
	for i := 1; i < attempt; i++ {
		d *= retryFactor
	}
	if d > retryCap {
		d = retryCap
	}
	jitter := 1 + (rand.Float64()*2-1)*jitterFactor //nolint:gosec // jitter, not crypto
	return time.Duration(float64(d) * jitter)
}

// mustConstantOne builds the {"values":{"0":{"constantValue":1}},"result":"0"}
// probe expression used by Session.Probe.
func mustConstantOne() *eeexpr.Expression {
	expr, err := eeexpr.Compile(eeexpr.Const(1))
	if err != nil {
		panic(err) // a constant cannot fail to compile
	}
	return expr
}

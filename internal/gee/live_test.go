package gee_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
	"github.com/SanTiwari07/NDVI_satellite/internal/gee"
	"github.com/SanTiwari07/NDVI_satellite/internal/gee/eeexpr"
	"github.com/SanTiwari07/NDVI_satellite/internal/pipeline"
)

// Live tests hit the real Earth Engine API. They are skipped unless
// GEE_LIVE_TEST=1, so `go test ./...` stays offline and deterministic.
//
// PRD §14 Phase 2 exit criterion: "a live smoke test computes s2_scene_count
// for the Nashik fixture and matches the Python number exactly."
func liveSession(t *testing.T) (*gee.Client, *config.Config) {
	t.Helper()
	if os.Getenv("GEE_LIVE_TEST") != "1" {
		t.Skip("set GEE_LIVE_TEST=1 to run live Earth Engine tests")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	sess, err := gee.NewSession(context.Background(), cfg)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	t.Logf("authenticated via %s, project %s", sess.Source, sess.ProjectID)
	return sess.NewClient(), cfg
}

func fixtureGeom() eeexpr.Geometry {
	return eeexpr.Polygon([][][]float64{{
		{73.79, 20.011},
		{73.7916, 20.011},
		{73.7916, 20.0122},
		{73.79, 20.0122},
		{73.79, 20.011},
	}})
}

func TestLiveProbe(t *testing.T) {
	c, cfg := liveSession(t)
	_ = cfg
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var got float64
	if err := c.ComputeValueNode(ctx, eeexpr.Const(1), &got); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if got != 1 {
		t.Errorf("probe returned %v, want 1", got)
	}
}

// TestLiveSceneCount is the Phase 2 acceptance gate. The pinned window matches
// legacy-python/tools/dump_graphs.py, so the number is stable and directly
// comparable with the Python client.
func TestLiveSceneCount(t *testing.T) {
	c, cfg := liveSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	b := eeexpr.NewBuilder()
	coll := pipeline.S2CollectionExpr(cfg, b, fixtureGeom(),
		pipeline.DateWindow{Start: "2025-01-01", End: "2025-04-01"})

	var count int
	if err := c.ComputeValueNode(ctx, coll.Size(), &count); err != nil {
		t.Fatalf("scene count: %v", err)
	}
	t.Logf("s2_scene_count = %d", count)
	if count <= 0 {
		t.Errorf("scene count = %d, expected a positive count for the Nashik fixture", count)
	}

	want := os.Getenv("GEE_EXPECTED_SCENE_COUNT")
	if want == "" {
		t.Log("set GEE_EXPECTED_SCENE_COUNT to assert against the Python number")
		return
	}
	if got := itoa(count); got != want {
		t.Errorf("scene count = %s, Python reported %s", got, want)
	}
}

// TestLiveGridSize exercises Geometry.coveringGrid + Projection.atScale, the
// pair most likely to be wrong in a hand-built graph.
func TestLiveGridSize(t *testing.T) {
	c, cfg := liveSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	var cells int
	grid := pipeline.GridExpr(cfg, fixtureGeom(), cfg.GridScaleM)
	if err := c.ComputeValueNode(ctx, grid.Size(), &cells); err != nil {
		t.Fatalf("grid size: %v", err)
	}
	t.Logf("grid cells at %dm = %d", cfg.GridScaleM, cells)
	if cells <= 0 {
		t.Errorf("grid size = %d, want > 0", cells)
	}
}

// TestLiveFarmMean drives the full S2 → indices → reduceRegion chain and
// checks every band comes back with a plausible value.
func TestLiveFarmMean(t *testing.T) {
	c, cfg := liveSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	b := eeexpr.NewBuilder()
	comp := pipeline.S2CollectionExpr(cfg, b, fixtureGeom(),
		pipeline.DateWindow{Start: "2025-01-01", End: "2025-04-01"}).Median()
	indexed := pipeline.ComputeAllIndices(cfg, comp)

	var means map[string]*float64
	if err := c.ComputeValueNode(ctx,
		pipeline.FarmMeanExpr(indexed, fixtureGeom(), pipeline.StatsBands), &means); err != nil {
		t.Fatalf("farm mean: %v", err)
	}
	for _, band := range pipeline.StatsBands {
		v, present := means[band]
		if !present {
			t.Errorf("band %s missing from reduceRegion result", band)
			continue
		}
		if v == nil {
			t.Logf("%s = null (masked)", band)
			continue
		}
		t.Logf("%s = %.6f", band, *v)
		if *v < -1.5 || *v > 1.5 {
			t.Errorf("%s = %v, outside the plausible [-1.5, 1.5] range for an index", band, *v)
		}
	}
}

// TestLiveCreateMap verifies the maps request shape and the returned tile URL
// template against §5.6.
func TestLiveCreateMap(t *testing.T) {
	c, cfg := liveSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	b := eeexpr.NewBuilder()
	comp := pipeline.S2CollectionExpr(cfg, b, fixtureGeom(),
		pipeline.DateWindow{Start: "2025-01-01", End: "2025-04-01"}).Median()
	indexed := pipeline.ComputeAllIndices(cfg, comp)
	smooth := pipeline.SmoothTileExpr(indexed, fixtureGeom(), "NDVI")

	url, err := c.CreateMap(ctx, smooth, gee.VisParams{
		Min: 0.0, Max: 1.0, Palette: cfg.NDVIPalette,
	})
	if err != nil {
		t.Fatalf("create map: %v", err)
	}
	t.Logf("tile url = %s", url)
	for _, want := range []string{
		"https://earthengine.googleapis.com/v1/projects/",
		"/maps/",
		"/tiles/{z}/{x}/{y}",
	} {
		if !contains(url, want) {
			t.Errorf("tile URL %q does not contain %q", url, want)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// TestLiveSessionSurvivesCallerContextCancellation is a regression test for a
// bug that only showed up after the server had been running for about an hour.
//
// oauth2.NewClient stores the context it is given and reuses it for every token
// refresh. cmd/server builds its session inside a startup probe bounded by a
// 60-second timeout with `defer cancel()`, so passing that context through left
// the client permanently cancelled once startup finished. The initial access
// token kept working, and then every refresh failed with
//
//	Post "https://oauth2.googleapis.com/token": context canceled
//
// losing Earth Engine access until the process restarted. It was invisible to
// every test that ran within the first hour of a fresh server, and was only
// caught by leaving the backend up and driving it from the frontend.
//
// Here the caller's context is cancelled immediately after the session is
// built, then a real API call is made with a fresh context. Before the fix this
// fails; after it, the refresh loop is detached and the call succeeds.
func TestLiveSessionSurvivesCallerContextCancellation(t *testing.T) {
	if os.Getenv("GEE_LIVE_TEST") != "1" {
		t.Skip("set GEE_LIVE_TEST=1 to run live Earth Engine tests")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}

	// Mirror cmd/server: a short-lived, cancellable startup context.
	startupCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	sess, err := gee.NewSession(startupCtx, cfg)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	cancel() // startup finishes — this must NOT disable the session

	client := sess.NewClient()
	callCtx, callCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer callCancel()

	var got float64
	if err := client.ComputeValueNode(callCtx, eeexpr.Const(1), &got); err != nil {
		t.Fatalf("call after the caller's context was cancelled: %v\n"+
			"the session must not capture a cancellable caller context", err)
	}
	if got != 1 {
		t.Errorf("probe returned %v, want 1", got)
	}
}

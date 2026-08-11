package pipeline

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
	"github.com/SanTiwari07/NDVI_satellite/internal/gee/eeexpr"
)

// These tests are the acceptance gate for Phase 2 (§5.4): no pipeline code is
// considered done until its expression graph matches the fixture captured from
// the real Python client. A wrong function or argument name produces a runtime
// 400 from Google rather than a compile error, so this is the only thing
// standing between the port and a silent outage.

const (
	fixStart = "2025-01-01"
	fixEnd   = "2025-04-01"
	fixDay   = "2025-02-14"
)

func fixtureCfg(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load("\x00nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	return cfg
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

func fixtureWindow() DateWindow { return DateWindow{Start: fixStart, End: fixEnd} }

func assertGraph(t *testing.T, name string, got eeexpr.Node) {
	t.Helper()

	expr, err := eeexpr.Compile(got)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	gotTree, err := eeexpr.CanonicaliseExpression(expr)
	if err != nil {
		t.Fatalf("canonicalise Go graph: %v", err)
	}

	path := filepath.Join("..", "gee", "eeexpr", "testdata", name+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture %s not present: %v", name, err)
	}
	wantTree, err := eeexpr.Canonicalise(raw)
	if err != nil {
		t.Fatalf("canonicalise fixture %s: %v", name, err)
	}

	if diff := cmp.Diff(wantTree, gotTree); diff != "" {
		t.Errorf("graph differs from fixture %s (-python +go):\n%s", name, truncate(diff, 4000))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n… (diff truncated)"
}

// ── G1-G4: acquisition ──────────────────────────────────────────────────────

func TestG01_S2Collection(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	assertGraph(t, "g01_s2_collection",
		S2CollectionExpr(cfg, b, fixtureGeom(), fixtureWindow()).N)
}

func TestG02_SceneCount(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	assertGraph(t, "g02_s2_scene_count",
		S2CollectionExpr(cfg, b, fixtureGeom(), fixtureWindow()).Size())
}

func TestG03_Composite(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	assertGraph(t, "g03_s2_composite",
		S2CollectionExpr(cfg, b, fixtureGeom(), fixtureWindow()).Median().N)
}

func TestG04_SCLMask(t *testing.T) {
	cfg := fixtureCfg(t)
	img := eeexpr.ImageLoad(cfg.Dataset + "/20250214T053001_20250214T053811_T43QCA")
	assertGraph(t, "g04_scl_mask", MaskCloudsSCL(cfg, img).N)
}

// ── G5/G6: indices ──────────────────────────────────────────────────────────

func TestG05_NormalizedDifferenceIndices(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	comp := S2CollectionExpr(cfg, b, fixtureGeom(), fixtureWindow()).Median()

	for _, tc := range []struct {
		fixture string
		got     eeexpr.Image
	}{
		{"g05_ndvi", NDVI(cfg, comp)},
		{"g05_ndmi", NDMI(cfg, comp)},
		{"g05_ndwi", NDWI(cfg, comp)},
		{"g05_gndvi", GNDVI(cfg, comp)},
	} {
		t.Run(tc.fixture, func(t *testing.T) { assertGraph(t, tc.fixture, tc.got.N) })
	}
}

// TestG06_ArithmeticRewrite checks EVI/SAVI against the arithmetic fixtures.
// They deliberately do NOT match g06_*_expression, which uses parseExpression
// (§7.3) — that divergence is the whole point of the rewrite.
func TestG06_ArithmeticRewrite(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	comp := S2CollectionExpr(cfg, b, fixtureGeom(), fixtureWindow()).Median()

	t.Run("evi", func(t *testing.T) { assertGraph(t, "g06_evi_arith", EVI(cfg, comp).N) })
	t.Run("savi", func(t *testing.T) { assertGraph(t, "g06_savi_arith", SAVI(cfg, comp).N) })
}

// TestG07_IndexedStackingOrder gates addBands ordering and the CVI weighted
// sum against the arithmetic-built fixture.
func TestG07_IndexedStackingOrder(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	comp := S2CollectionExpr(cfg, b, fixtureGeom(), fixtureWindow()).Median()
	assertGraph(t, "g07_indexed_arith", ComputeAllIndices(cfg, comp).N)
}

// ── G8-G12: grid and statistics ─────────────────────────────────────────────

func TestG08_CoveringGrid(t *testing.T) {
	cfg := fixtureCfg(t)
	assertGraph(t, "g08_grid", GridExpr(cfg, fixtureGeom(), cfg.GridScaleM).N)
	assertGraph(t, "g08_grid_size", GridExpr(cfg, fixtureGeom(), cfg.GridScaleM).Size())
}

func TestG09_GridReduction(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	comp := S2CollectionExpr(cfg, b, fixtureGeom(), fixtureWindow()).Median()
	indexed := ComputeAllIndices(cfg, comp)
	grid := GridExpr(cfg, fixtureGeom(), cfg.GridScaleM)
	// The fixture was built from the parseExpression-based indexed image, so
	// only the reduction wrapper is comparable; the band subset and reducer
	// arguments are what this asserts.
	got := ReduceGridExpr(cfg, b, indexed, grid, IndexBands)
	assertGraph(t, "g09_grid_reduced_arith", got.N)
}

func TestG10_FarmMean(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	comp := S2CollectionExpr(cfg, b, fixtureGeom(), fixtureWindow()).Median()
	indexed := ComputeAllIndices(cfg, comp)
	assertGraph(t, "g10_farm_mean_arith",
		FarmMeanExpr(indexed, fixtureGeom(), StatsBands))
}

func TestG11_CVIStdDev(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	comp := S2CollectionExpr(cfg, b, fixtureGeom(), fixtureWindow()).Median()
	indexed := ComputeAllIndices(cfg, comp)
	assertGraph(t, "g11_cvi_stddev_arith", CVIStdDevExpr(indexed, fixtureGeom()))
}

func TestG12_NDVIHistogram(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	comp := S2CollectionExpr(cfg, b, fixtureGeom(), fixtureWindow()).Median()
	indexed := ComputeAllIndices(cfg, comp)
	assertGraph(t, "g12_ndvi_histogram_arith", NDVIHistogramExpr(indexed, fixtureGeom()))
}

// ── G13-G16 ─────────────────────────────────────────────────────────────────

func TestG13_SmoothTile(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	comp := S2CollectionExpr(cfg, b, fixtureGeom(), fixtureWindow()).Median()
	indexed := ComputeAllIndices(cfg, comp)
	assertGraph(t, "g13_smooth_tile_ndvi_arith",
		SmoothTileExpr(indexed, fixtureGeom(), "NDVI").N)
}

func TestG14_AvailableDates(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	assertGraph(t, "g14_available_dates",
		AvailableDatesExpr(cfg, b, fixtureGeom(), fixtureWindow()))
}

func TestG15_SingleDay(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	w, err := SingleDayWindow(fixDay)
	if err != nil {
		t.Fatal(err)
	}
	if w.Start != "2025-02-14" || w.End != "2025-02-15" {
		t.Fatalf("single-day window = %+v, want [2025-02-14, 2025-02-15)", w)
	}
	coll := S2CollectionExpr(cfg, b, fixtureGeom(), w)
	assertGraph(t, "g15_single_day_collection", coll.N)
	assertGraph(t, "g15_single_day_composite", coll.Median().N)
}

func TestG16_PointSample(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	comp := S2CollectionExpr(cfg, b, fixtureGeom(), fixtureWindow()).Median()
	indexed := ComputeAllIndices(cfg, comp)
	assertGraph(t, "g16_point_sample_arith",
		PointSampleExpr(indexed, 73.7908, 20.0116, "NDVI", 10))
}

// ── G17-G24: Sentinel-1 radar ───────────────────────────────────────────────

const fixRadarDay = "2025-02-14"

func fixtureS1Window(t *testing.T, cfg *config.Config) DateWindow {
	t.Helper()
	w, err := S1CompositeWindow(cfg, fixRadarDay, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestG17_S1BaseCollection(t *testing.T) {
	cfg := fixtureCfg(t)
	assertGraph(t, "g17_s1_base_collection", BaseS1CollectionExpr(cfg, eeexpr.NewBuilder(), fixtureGeom()).N)
}

func TestG17_S1AvailableDates(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	assertGraph(t, "g17_s1_available_dates",
		S1AvailableDatesExpr(cfg, b, fixtureGeom(), fixtureWindow()))
}

func TestG18_SpeckleFilterAndComposite(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	w := fixtureS1Window(t, cfg)
	if w.Start != "2025-02-08" || w.End != "2025-02-21" {
		t.Fatalf("S1 window = %+v, want [2025-02-08, 2025-02-21)", w)
	}
	coll := S1CollectionExpr(cfg, b, fixtureGeom(), w)
	assertGraph(t, "g18_s1_speckle_collection", coll.N)
	assertGraph(t, "g18_s1_scene_count", coll.Size())
	assertGraph(t, "g18_s1_composite", coll.Median().N)
}

func TestG19_G21_RadarIndices(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	comp := S1CollectionExpr(cfg, b, fixtureGeom(), fixtureS1Window(t, cfg)).Median()
	vv, vh := comp.Select("VV"), comp.Select("VH")

	t.Run("smi", func(t *testing.T) { assertGraph(t, "g19_smi", SMI(cfg, vv).N) })
	t.Run("rvi", func(t *testing.T) { assertGraph(t, "g20_rvi", RVI(vv, vh).N) })
	t.Run("ratio", func(t *testing.T) { assertGraph(t, "g21_ratio", RATIO(vv, vh).N) })
	t.Run("indexed", func(t *testing.T) {
		assertGraph(t, "g21_radar_indexed", ComputeRadarIndices(cfg, comp).N)
	})
}

func TestG22_SmoothRadarTile(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	comp := S1CollectionExpr(cfg, b, fixtureGeom(), fixtureS1Window(t, cfg)).Median()
	radar := ComputeRadarIndices(cfg, comp)
	assertGraph(t, "g22_smooth_radar_tile_smi",
		SmoothRadarTileExpr(radar, fixtureGeom(), "SMI").N)
}

func TestG23_RadarGridReduction(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	comp := S1CollectionExpr(cfg, b, fixtureGeom(), fixtureS1Window(t, cfg)).Median()
	radar := ComputeRadarIndices(cfg, comp)
	grid := GridExpr(cfg, fixtureGeom(), cfg.GridScaleM)
	assertGraph(t, "g23_radar_grid_reduced", ReduceRadarGridExpr(cfg, b, radar, grid).N)
}

func TestG24_RadarFarmMean(t *testing.T) {
	cfg := fixtureCfg(t)
	b := eeexpr.NewBuilder()
	comp := S1CollectionExpr(cfg, b, fixtureGeom(), fixtureS1Window(t, cfg)).Median()
	radar := ComputeRadarIndices(cfg, comp)
	assertGraph(t, "g24_radar_farm_mean", RadarFarmMeanExpr(cfg, radar, fixtureGeom()))
}

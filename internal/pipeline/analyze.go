package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
	"github.com/SanTiwari07/NDVI_satellite/internal/gee"
	"github.com/SanTiwari07/NDVI_satellite/internal/gee/eeexpr"
)

// EEClient is the slice of the Earth Engine client this package needs.
//
// Declaring it as an interface here keeps internal/pipeline free of any direct
// HTTP concern (§3.2) and makes the orchestration unit-testable with a fake.
type EEClient interface {
	ComputeValueNode(ctx context.Context, n eeexpr.Node, out any) error
	ComputeFeatures(ctx context.Context, expr *eeexpr.Expression, maxFeatures int) (*gee.FeatureCollection, error)
	CreateMap(ctx context.Context, img eeexpr.Image, vis gee.VisParams) (string, error)
}

// Analyzer orchestrates the Sentinel-2 and Sentinel-1 pipelines.
type Analyzer struct {
	Cfg   *config.Config
	EE    EEClient
	Cache *ExprCache
	Log   *slog.Logger
	// Now is injectable so tests can pin the date window. Defaults to
	// time.Now, which is LOCAL time on purpose: datetime.date.today() uses the
	// server's timezone and the window rolls over at local midnight (§13.8).
	Now func() time.Time
}

func (a *Analyzer) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a *Analyzer) log() *slog.Logger {
	if a.Log != nil {
		return a.Log
	}
	return slog.Default()
}

// AnalyzeResult is the /api/analyze response body (§10.3).
//
// The embedded GridCollection supplies "type" and "features"; the remaining
// fields are added at the top level exactly as the Python does.
type AnalyzeResult struct {
	Type         string             `json:"type"`
	Features     []GridFeature      `json:"features"`
	FarmSummary  FarmSummary        `json:"farm_summary"`
	FarmBoundary json.RawMessage    `json:"farm_boundary"`
	NDVITileURL  *string            `json:"ndvi_tile_url"`
	TileURL      *string            `json:"tile_url"`
	IndexTiles   map[string]*string `json:"index_tiles"`
}

// vegetationTileBands is the set /api/analyze generates tiles for, in order.
var vegetationTileBands = []string{"NDVI", "EVI", "SAVI", "NDMI", "NDWI", "GNDVI"}

// Analyze runs the full Sentinel-2 pipeline (E2).
//
// Returns (nil, nil) when no imagery is found, which the handler turns into the
// HTTP 200 + {"error": …} body of §13.2.
func (a *Analyzer) Analyze(ctx context.Context, geometry json.RawMessage) (*AnalyzeResult, error) {
	geom, err := GeometryFromGeoJSON(geometry)
	if err != nil {
		return nil, err
	}
	window := LookbackWindow(a.Cfg, a.now())
	b := eeexpr.NewBuilder()
	coll := S2CollectionExpr(a.Cfg, b, geom, window)

	started := time.Now()
	var sceneCount int
	if err := a.EE.ComputeValueNode(ctx, coll.Size(), &sceneCount); err != nil {
		return nil, fmt.Errorf("scene count: %w", err)
	}
	a.log().Info(fmt.Sprintf("Scenes found after filtering: %d", sceneCount),
		"stage", "scene_count", "ms", time.Since(started).Milliseconds())

	if sceneCount == 0 {
		a.log().Warn("No clean Sentinel-2 scenes found. " +
			"Try widening LOOKBACK_DAYS or MAX_CLOUD_COVER_PCT in config.py.")
		return nil, nil
	}

	composite := coll.Median()
	indexed := ComputeAllIndices(a.Cfg, composite)

	grid, _, err := a.buildGrid(ctx, geom)
	if err != nil {
		return nil, err
	}

	features, err := a.reduceGrid(ctx, b, indexed, grid, IndexBands)
	if err != nil {
		return nil, err
	}
	features = a.postProcessVegetation(features)

	summary := a.farmStatistics(ctx, indexed, geom, coll, sceneCount)

	result := &AnalyzeResult{
		Type:         "FeatureCollection",
		Features:     features,
		FarmSummary:  summary,
		FarmBoundary: geometry,
		IndexTiles:   map[string]*string{},
	}

	// Tile URLs. Any individual failure yields null for that layer rather than
	// failing the whole request — the Python catches and returns None (§10.3).
	for _, band := range vegetationTileBands {
		url := a.tileURL(ctx, indexed, geom, band, gee.VisParams{
			Min: 0.0, Max: 1.0, Palette: a.Cfg.NDVIPalette,
		}, false)
		result.IndexTiles[lower(band)+"_tile_url"] = url
	}
	cviURL := a.tileURL(ctx, indexed, geom, "CVI", gee.VisParams{
		Min: 0.0, Max: 1.0, Palette: a.Cfg.CVIPalette,
	}, false)
	result.IndexTiles["cvi_tile_url"] = cviURL

	// ndvi_tile_url is duplicated at the top level AND inside index_tiles;
	// tile_url is the CVI layer. Both duplications are preserved (§10.3).
	result.NDVITileURL = result.IndexTiles["ndvi_tile_url"]
	result.TileURL = cviURL

	a.Cache.Put(CachedAnalysis{Image: indexed, Geometry: geom},
		CacheKey(geometry, "", "s2"), LastKey)

	a.log().Info(fmt.Sprintf("Analysis complete — %d scenes, %d grid cells, confidence=%.4f",
		sceneCount, len(features), summary.Confidence))
	return result, nil
}

// AnalyzeDayResult is the /api/analyze-day response (§10.5).
//
// It carries date and scene_count, and ONLY ndvi_tile_url — no tile_url and no
// index_tiles.
type AnalyzeDayResult struct {
	Type         string          `json:"type"`
	Features     []GridFeature   `json:"features"`
	FarmSummary  FarmSummary     `json:"farm_summary"`
	FarmBoundary json.RawMessage `json:"farm_boundary"`
	NDVITileURL  *string         `json:"ndvi_tile_url"`
	Date         string          `json:"date"`
	SceneCount   int             `json:"scene_count"`
}

// AnalyzeDay runs the single-date pipeline (E4).
func (a *Analyzer) AnalyzeDay(ctx context.Context, geometry json.RawMessage, date string) (*AnalyzeDayResult, error) {
	geom, err := GeometryFromGeoJSON(geometry)
	if err != nil {
		return nil, err
	}
	window, err := SingleDayWindow(date)
	if err != nil {
		return nil, err
	}

	b := eeexpr.NewBuilder()
	coll := S2CollectionExpr(a.Cfg, b, geom, window)

	var sceneCount int
	if err := a.EE.ComputeValueNode(ctx, coll.Size(), &sceneCount); err != nil {
		return nil, fmt.Errorf("scene count: %w", err)
	}
	if sceneCount == 0 {
		return nil, nil
	}

	indexed := ComputeAllIndices(a.Cfg, coll.Median())

	grid, _, err := a.buildGrid(ctx, geom)
	if err != nil {
		return nil, err
	}
	features, err := a.reduceGrid(ctx, b, indexed, grid, IndexBands)
	if err != nil {
		return nil, err
	}
	features = a.postProcessVegetation(features)

	// collection is nil here, so avg_cloud falls back to MAX_CLOUD_COVER_PCT
	// (= 20) — /api/analyze-day always uses 20 (§7.8, §10.5).
	summary := a.farmStatistics(ctx, indexed, geom, eeexpr.Collection{}, sceneCount)

	a.Cache.Put(CachedAnalysis{Image: indexed, Geometry: geom},
		CacheKey(geometry, date, "s2-day"), LastKey)

	return &AnalyzeDayResult{
		Type:         "FeatureCollection",
		Features:     features,
		FarmSummary:  summary,
		FarmBoundary: geometry,
		NDVITileURL: a.tileURL(ctx, indexed, geom, "NDVI", gee.VisParams{
			Min: 0.0, Max: 1.0, Palette: a.Cfg.NDVIPalette,
		}, false),
		Date:       date,
		SceneCount: sceneCount,
	}, nil
}

// AvailableDates lists distinct S2 acquisition dates (E3).
func (a *Analyzer) AvailableDates(ctx context.Context, geometry json.RawMessage) ([]string, error) {
	geom, err := GeometryFromGeoJSON(geometry)
	if err != nil {
		return nil, err
	}
	b := eeexpr.NewBuilder()
	expr := AvailableDatesExpr(a.Cfg, b, geom, LookbackWindow(a.Cfg, a.now()))

	dates := []string{}
	if err := a.EE.ComputeValueNode(ctx, expr, &dates); err != nil {
		return nil, err
	}
	a.log().Info(fmt.Sprintf("Found %d available dates for the polygon.", len(dates)))
	return dates, nil
}

// Sample reads a single pixel from the cached analysis (E7).
//
// Returns (nil, false) when nothing is cached, which the handler turns into the
// 404 of §10.7.
func (a *Analyzer) Sample(ctx context.Context, lat, lng float64, band string) (*float64, bool) {
	cached, ok := a.Cache.Get(LastKey)
	if !ok {
		return nil, false
	}

	expr := PointSampleExpr(cached.Image, lng, lat, band, float64(a.Cfg.GridScaleM))
	var out map[string]*float64
	if err := a.EE.ComputeValueNode(ctx, expr, &out); err != nil {
		// sample_point_value catches every exception and returns None, so a
		// failure here is a null value, not an HTTP error.
		a.log().Error(fmt.Sprintf("Point sampling failed at (%.4f, %.4f): %v", lat, lng, err))
		return nil, true
	}
	v := out[band]
	if v == nil {
		return nil, true
	}
	r := Round4(*v)
	return &r, true
}

// ── shared helpers ──────────────────────────────────────────────────────────

// buildGrid runs the auto-coarsening search and returns the final grid.
func (a *Analyzer) buildGrid(ctx context.Context, geom eeexpr.Geometry) (eeexpr.Collection, int, error) {
	var lastErr error
	scale, cells, err := GridScaleSearch(a.Cfg, func(scale int) (int, error) {
		var n int
		if err := a.EE.ComputeValueNode(ctx, GridExpr(a.Cfg, geom, scale).Size(), &n); err != nil {
			lastErr = err
			return 0, err
		}
		a.log().Info(fmt.Sprintf("Grid at %dm: %d cells", scale, n))
		return n, nil
	})
	if err != nil {
		return eeexpr.Collection{}, 0, fmt.Errorf("grid sizing: %w", lastErr)
	}
	a.log().Info(fmt.Sprintf("Final grid: scale=%dm, cells=%d (max allowed: %d)",
		scale, cells, a.Cfg.MaxGridCells))
	return GridExpr(a.Cfg, geom, scale), cells, nil
}

// reduceGrid materialises the per-cell reduction and rounds to 4 dp.
func (a *Analyzer) reduceGrid(
	ctx context.Context,
	b *eeexpr.Builder,
	img eeexpr.Image,
	grid eeexpr.Collection,
	bands []string,
) ([]GridFeature, error) {
	reduced := ReduceGridExpr(a.Cfg, b, img, grid, bands)
	expr, err := eeexpr.Compile(reduced.N)
	if err != nil {
		return nil, err
	}
	// Guard at 2× MaxGridCells, per §11.2.
	fc, err := a.EE.ComputeFeatures(ctx, expr, a.Cfg.MaxGridCells*2)
	if err != nil {
		return nil, fmt.Errorf("grid reduction: %w", err)
	}

	out := make([]GridFeature, 0, len(fc.Features))
	for _, f := range fc.Features {
		props := make(map[string]any, len(bands)+1)
		for _, band := range bands {
			// Lower-case the band name into the output property key.
			key := lower(band)
			v := floatOrNil(f.Properties[band])
			if v == nil {
				props[key] = nil // must stay JSON null, never 0 (§13.1)
				continue
			}
			props[key] = Round4(*v)
		}
		out = append(out, GridFeature{
			Type:       "Feature",
			Geometry:   withEEGeometryFields(f.Geometry),
			Properties: props,
		})
	}
	return out, nil
}

// withEEGeometryFields adds the two keys the Python client attaches to every
// geometry it returns from getInfo but which table:computeFeatures omits:
//
//	"geodesic": false
//	"crs": {"type": "name", "properties": {"name": "EPSG:4326"}}
//
// The coordinates are byte-identical either way, so this is purely about the
// frozen contract (§0.1): the frontend receives these keys today and the
// response must not quietly lose them. EPSG:4326 is hardcoded because GridExpr
// always tiles in that projection.
func withEEGeometryFields(raw json.RawMessage) json.RawMessage {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw
	}
	if _, present := m["geodesic"]; !present {
		m["geodesic"] = false
	}
	if _, present := m["crs"]; !present {
		m["crs"] = map[string]any{
			"type":       "name",
			"properties": map[string]any{"name": "EPSG:4326"},
		}
	}
	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return out
}

// postProcessVegetation applies the smoothing pass and the interpretation
// label, in the order grid_service does.
func (a *Analyzer) postProcessVegetation(features []GridFeature) []GridFeature {
	// CVI is EXCLUDED from smoothing — it is derived, not measured (§7.5).
	smoothBands := make([]string, 0, len(IndexBands)-1)
	for _, b := range IndexBands {
		if b == "CVI" {
			continue
		}
		smoothBands = append(smoothBands, lower(b))
	}
	features = SmoothGridValues(features, smoothBands, a.Cfg.SmoothSigmaFactor)

	// Interpretation is attached from the UNSMOOTHED cvi (§7.5 step 4).
	for i := range features {
		cvi := floatOrNil(features[i].Properties["cvi"])
		features[i].Properties["interpretation"] =
			config.Interpret(cvi, a.Cfg.CVIThresholds,
				"No data available", "Poor vegetation, needs attention")
	}
	return features
}

// farmStatistics reproduces extract_farm_statistics, including its fallback.
func (a *Analyzer) farmStatistics(
	ctx context.Context,
	indexed eeexpr.Image,
	geom eeexpr.Geometry,
	coll eeexpr.Collection,
	sceneCount int,
) FarmSummary {
	// Step 1: average cloud cover (§7.8). The Python is
	//
	//   try:
	//       avg_cloud = coll.aggregate_mean(...).getInfo() if collection else MAX
	//   except Exception:
	//       avg_cloud = MAX
	//   ...
	//   confidence = compute_confidence(scene_count, avg_cloud or 0, cvi_std)
	//
	// so there are THREE distinct outcomes, and conflating them changes the
	// score:
	//
	//   no collection            → MAX_CLOUD_COVER_PCT (20)
	//   call raises              → MAX_CLOUD_COVER_PCT (20)
	//   call returns null or 0   → 0, because `None or 0` is 0
	//
	// K13: the third case is what actually happens on /api/analyze. The two
	// .map() calls strip image properties, so CLOUDY_PIXEL_PERCENTAGE is gone
	// by the time aggregate_mean runs and it returns null — meaning the cloud
	// term contributes a constant 0.30 and is effectively dead. Verified
	// against the running Python backend. Preserved, not fixed (§0.8).
	avgCloud := float64(a.Cfg.MaxCloudCoverPct)
	if coll.N != nil {
		var v *float64
		if err := a.EE.ComputeValueNode(ctx,
			coll.AggregateMean("CLOUDY_PIXEL_PERCENTAGE"), &v); err != nil {
			avgCloud = float64(a.Cfg.MaxCloudCoverPct) // the except branch
		} else if v == nil {
			avgCloud = 0 // `None or 0`
		} else {
			avgCloud = *v
		}
	}

	means := map[string]*float64{}
	var cviStd float64
	histogram := EmptyHistogram()

	// Steps 2-4 share one failure envelope: if ANY of them fails, all means
	// become null, cviStd becomes 0 and the histogram all-zero — and the
	// confidence score is still computed (§7.8 step 6).
	failed := false

	if err := a.EE.ComputeValueNode(ctx,
		FarmMeanExpr(indexed, geom, StatsBands), &means); err != nil {
		a.log().Error("Failed to extract farm stats: " + err.Error())
		failed = true
	}

	if !failed {
		var std map[string]*float64
		if err := a.EE.ComputeValueNode(ctx, CVIStdDevExpr(indexed, geom), &std); err != nil {
			a.log().Error("Failed to extract farm stats: " + err.Error())
			failed = true
		} else if v := std["CVI"]; v != nil {
			cviStd = *v
		}
	}

	if !failed {
		var hist struct {
			Groups []map[string]any `json:"groups"`
		}
		if err := a.EE.ComputeValueNode(ctx, NDVIHistogramExpr(indexed, geom), &hist); err != nil {
			a.log().Error("Failed to extract farm stats: " + err.Error())
			failed = true
		} else {
			histogram = HistogramFromGroups(hist.Groups)
		}
	}

	if failed {
		means = map[string]*float64{}
		for _, b := range StatsBands {
			means[b] = nil
		}
		cviStd = 0.0
		histogram = EmptyHistogram()
	}

	return BuildFarmSummary(a.Cfg, sceneCount, avgCloud, cviStd, means, histogram)
}

// tileURL builds one tile layer, returning nil on failure rather than
// propagating the error (§10.3: "Any individual tile URL may be null").
func (a *Analyzer) tileURL(
	ctx context.Context,
	img eeexpr.Image,
	geom eeexpr.Geometry,
	band string,
	vis gee.VisParams,
	radar bool,
) *string {
	var smooth eeexpr.Image
	if radar {
		smooth = SmoothRadarTileExpr(img, geom, band)
	} else {
		smooth = SmoothTileExpr(img, geom, band)
	}

	url, err := a.EE.CreateMap(ctx, smooth, vis)
	if err != nil {
		a.log().Error(fmt.Sprintf("Failed to generate smooth tile URL for %s: %v", band, err))
		return nil
	}
	a.log().Info("Smooth tile URL generated for band=" + band)
	return &url
}

// GeometryFromGeoJSON converts a validated GeoJSON polygon into an EE geometry.
func GeometryFromGeoJSON(raw json.RawMessage) (eeexpr.Geometry, error) {
	var g struct {
		Type        string          `json:"type"`
		Coordinates [][][]float64   `json:"coordinates"`
		Raw         json.RawMessage `json:"-"`
	}
	if err := json.Unmarshal(raw, &g); err != nil {
		return eeexpr.Geometry{}, fmt.Errorf(
			"Could not create GEE geometry from provided GeoJSON: %w", err)
	}
	if len(g.Coordinates) == 0 {
		return eeexpr.Geometry{}, fmt.Errorf(
			"Could not create GEE geometry from provided GeoJSON: empty coordinates")
	}
	return eeexpr.Polygon(g.Coordinates), nil
}

func lower(s string) string {
	out := []byte(s)
	for i := range out {
		if out[i] >= 'A' && out[i] <= 'Z' {
			out[i] += 'a' - 'A'
		}
	}
	return string(out)
}

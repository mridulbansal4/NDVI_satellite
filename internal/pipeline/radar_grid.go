package pipeline

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SanTiwari07/NDVI_satellite/internal/gee"
	"github.com/SanTiwari07/NDVI_satellite/internal/gee/eeexpr"
)

// RadarSummary is the radar_summary object of §10.6. Key names are exact.
type RadarSummary struct {
	SceneCount             int      `json:"scene_count"`
	VVMean                 *float64 `json:"vv_mean"`
	VHMean                 *float64 `json:"vh_mean"`
	VVVHRatio              *float64 `json:"vv_vh_ratio"`
	RVI                    *float64 `json:"rvi"`
	SMI                    *float64 `json:"smi"`
	MoistureClassification string   `json:"moisture_classification"`
}

// RadarResult is the /api/analyze-radar response body (§10.6).
type RadarResult struct {
	Type         string             `json:"type"`
	Features     []GridFeature      `json:"features"`
	RadarSummary RadarSummary       `json:"radar_summary"`
	FarmBoundary json.RawMessage    `json:"farm_boundary"`
	// Date echoes the request value and is null when the client omitted it.
	Date        *string            `json:"date"`
	SceneCount  int                `json:"scene_count"`
	RadarTiles  map[string]*string `json:"radar_tiles"`
	TileURL     *string            `json:"tile_url"`
}

// AnalyzeRadar runs the Sentinel-1 pipeline (E6).
//
// Returns (nil, nil) when no imagery is found; the handler turns that into the
// 200-with-error body, choosing between the two wordings of §10.6.
func (a *Analyzer) AnalyzeRadar(ctx context.Context, geometry json.RawMessage, date *string) (*RadarResult, error) {
	geom, err := GeometryFromGeoJSON(geometry)
	if err != nil {
		return nil, err
	}
	target := ""
	if date != nil {
		target = *date
	}
	window, err := S1CompositeWindow(a.Cfg, target, a.now())
	if err != nil {
		return nil, err
	}

	b := eeexpr.NewBuilder()
	coll := S1CollectionExpr(a.Cfg, b, geom, window)

	var sceneCount int
	if err := a.EE.ComputeValueNode(ctx, coll.Size(), &sceneCount); err != nil {
		return nil, fmt.Errorf("S1 scene count: %w", err)
	}
	a.log().Info(fmt.Sprintf("S1 scenes found after filtering: %d", sceneCount))
	if sceneCount == 0 {
		a.log().Warn("No Sentinel-1 scenes for this window. " +
			"Try a different date or widen S1_DATE_WINDOW_DAYS in config.py.")
		return nil, nil
	}

	radar := ComputeRadarIndices(a.Cfg, coll.Median())

	grid, _, err := a.buildGrid(ctx, geom)
	if err != nil {
		return nil, err
	}
	features, err := a.reduceGrid(ctx, b, radar, grid, a.Cfg.RadarBands)
	if err != nil {
		return nil, err
	}

	// Unlike the vegetation path, ALL five radar bands are smoothed (§7.9).
	smoothBands := make([]string, len(a.Cfg.RadarBands))
	for i, band := range a.Cfg.RadarBands {
		smoothBands[i] = lower(band)
	}
	features = SmoothGridValues(features, smoothBands, a.Cfg.SmoothSigmaFactor)

	// Moisture classification comes from the SMOOTHED smi — again unlike the
	// vegetation path, which labels from the unsmoothed cvi.
	for i := range features {
		smi := floatOrNil(features[i].Properties["smi"])
		features[i].Properties["moisture_classification"] = ClassifyMoisture(a.Cfg, smi)
	}

	summary := a.radarStatistics(ctx, radar, geom, sceneCount)

	result := &RadarResult{
		Type:         "FeatureCollection",
		Features:     features,
		RadarSummary: summary,
		FarmBoundary: geometry,
		Date:         date,
		SceneCount:   sceneCount,
		RadarTiles:   map[string]*string{},
	}

	for _, band := range a.Cfg.RadarBands {
		bounds := a.Cfg.RadarVisBounds[band]
		// Moisture-oriented layers use the red→green ramp; RVI and RATIO use a
		// distinct sequential palette so they do not read as moisture.
		palette := a.Cfg.RadarSequentialPalette
		if band == "SMI" || band == "VV" || band == "VH" {
			palette = a.Cfg.RadarMoisturePalette
		}
		result.RadarTiles[lower(band)+"_tile_url"] = a.tileURL(ctx, radar, geom, band,
			gee.VisParams{Min: bounds.Min, Max: bounds.Max, Palette: palette}, true)
	}
	result.TileURL = result.RadarTiles["smi_tile_url"]

	// K10: app._last_radar_image is written but never read by any handler.
	// The write is preserved so behaviour matches, and so a future hover
	// endpoint has the same data available.
	// parity: unused by any handler, mirrors app.py
	a.Cache.Put(CachedAnalysis{Image: radar, Geometry: geom},
		CacheKey(geometry, target, "s1"), LastRadarKey)

	a.log().Info(fmt.Sprintf("Radar analysis complete — %d scenes, %d cells, SMI=%v (%s)",
		sceneCount, len(features), derefOrNil(summary.SMI), summary.MoistureClassification))
	return result, nil
}

// RadarAvailableDates lists distinct S1 acquisition dates (E5).
func (a *Analyzer) RadarAvailableDates(ctx context.Context, geometry json.RawMessage) ([]string, error) {
	geom, err := GeometryFromGeoJSON(geometry)
	if err != nil {
		return nil, err
	}
	b := eeexpr.NewBuilder()
	expr := S1AvailableDatesExpr(a.Cfg, b, geom, S1LookbackWindow(a.Cfg, a.now()))

	dates := []string{}
	if err := a.EE.ComputeValueNode(ctx, expr, &dates); err != nil {
		return nil, err
	}
	a.log().Info(fmt.Sprintf("Available S1 dates: %d unique", len(dates)))
	return dates, nil
}

// radarStatistics reproduces extract_radar_statistics, including its fallback
// to all-null means on failure.
func (a *Analyzer) radarStatistics(
	ctx context.Context,
	radar eeexpr.Image,
	geom eeexpr.Geometry,
	sceneCount int,
) RadarSummary {
	means := map[string]*float64{}
	if err := a.EE.ComputeValueNode(ctx,
		RadarFarmMeanExpr(a.Cfg, radar, geom), &means); err != nil {
		a.log().Error("Failed to extract radar stats: " + err.Error())
		means = map[string]*float64{}
		for _, b := range a.Cfg.RadarBands {
			means[b] = nil
		}
	}

	r := func(key string) *float64 {
		v := means[key]
		if v == nil {
			return nil
		}
		out := Round4(*v)
		return &out
	}

	smi := r("SMI")
	return RadarSummary{
		SceneCount:             sceneCount,
		VVMean:                 r("VV"),
		VHMean:                 r("VH"),
		VVVHRatio:              r("RATIO"),
		RVI:                    r("RVI"),
		SMI:                    smi,
		MoistureClassification: ClassifyMoisture(a.Cfg, smi),
	}
}

func derefOrNil(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

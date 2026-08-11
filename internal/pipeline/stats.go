package pipeline

import (
	"math"
	"strconv"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
	"github.com/SanTiwari07/NDVI_satellite/internal/gee/eeexpr"
)

// FarmMeanExpr is the farm-wide mean over the seven index bands (G10).
// scale 10, maxPixels 1e9.
func FarmMeanExpr(img eeexpr.Image, geom eeexpr.Geometry, bands []string) eeexpr.Node {
	return img.Select(bands...).ReduceRegion(eeexpr.ReducerMean(), geom.N, 10, 1e9)
}

// CVIStdDevExpr is the CVI standard deviation used by the confidence score (G11).
func CVIStdDevExpr(img eeexpr.Image, geom eeexpr.Geometry) eeexpr.Node {
	return img.Select("CVI").ReduceRegion(eeexpr.ReducerStdDev(), geom.N, 10, 1e9)
}

// NDVIHistogramExpr is the 20-bucket NDVI area histogram (G12).
//
//	clamp to [0, 0.9999] → ÷0.05 → floor → int  ⇒ bucket index 0..19
//	pixelArea().addBands(bucket) → reduceRegion(sum().group(1, "bucket"))
//
// The upper clamp is 0.9999 rather than 1.0 precisely so that NDVI == 1 lands
// in bucket 19 instead of a 21st bucket.
func NDVIHistogramExpr(img eeexpr.Image, geom eeexpr.Geometry) eeexpr.Node {
	bucket := img.Select("NDVI").
		MaxNum(0.0).
		MinNum(0.9999).
		DivideNum(0.05).
		Floor().
		Int()
	return eeexpr.PixelArea().AddBands(bucket).ReduceRegion(
		eeexpr.ReducerSum().Group(1, "bucket"),
		geom.N,
		10,
		1e9,
	)
}

// ── Client-side post-processing ─────────────────────────────────────────────

// Round4 mirrors Python's round(x, 4).
//
// Python uses banker's rounding on exact .5 ties; those are vanishingly rare in
// float data and well inside the 1e-6 parity bar. Do not "improve" this without
// a test (§11.3).
func Round4(v float64) float64 { return math.Round(v*1e4) / 1e4 }

// Round2 mirrors round(x, 2), used by the histogram's hectare conversion.
func Round2(v float64) float64 { return math.Round(v*1e2) / 1e2 }

// ConfidenceScore reproduces compute_confidence (§7.8).
//
//	scene_score = min(scene_count/5, 1)
//	cloud_score = max(0, 1 − avg_cloud/100)
//	std_score   = max(0, 1 − cvi_std/0.3)
//	confidence  = round(clamp(0.50·scene + 0.30·cloud + 0.20·std, 0, 1), 4)
//
// K1: the result is 0-1 here, while schema.sql documents vi_reports
// .confidence_score as 0-100 and the README prints "56.46%". The frontend does
// the formatting; this is NOT corrected during the migration.
func ConfidenceScore(cfg *config.Config, sceneCount int, avgCloudPct, cviStd float64) float64 {
	sceneScore := math.Min(float64(sceneCount)/float64(cfg.ConfidenceSceneTarget), 1.0)
	cloudScore := math.Max(0.0, 1.0-avgCloudPct/100.0)
	stdScore := math.Max(0.0, 1.0-cviStd/cfg.ConfidenceStdMax)

	c := 0.50*sceneScore + 0.30*cloudScore + 0.20*stdScore
	return Round4(math.Min(math.Max(c, 0.0), 1.0))
}

// IndexSummary is one entry of farm_summary.indices.
//
// Mean is a POINTER: a masked or failed reduction must serialise as JSON null,
// never 0. A cell that emits 0 instead of null renders dark red on the heatmap
// and is a visible, user-facing bug (§13.1).
type IndexSummary struct {
	Mean           *float64 `json:"mean"`
	Interpretation string   `json:"interpretation"`
}

// FarmSummary is the farm_summary object of §10.3.
type FarmSummary struct {
	Confidence    float64                 `json:"confidence"`
	SceneCount    int                     `json:"scene_count"`
	Indices       map[string]IndexSummary `json:"indices"`
	NDVIHistogram map[string]float64      `json:"ndvi_histogram"`
}

// EmptyHistogram returns the all-zero 20-bucket histogram. Keys are STRINGS
// "0".."19" and all twenty are always present (§10.3).
func EmptyHistogram() map[string]float64 {
	h := make(map[string]float64, 20)
	for i := 0; i < 20; i++ {
		h[strconv.Itoa(i)] = 0.0
	}
	return h
}

// HistogramFromGroups converts reduceRegion's grouped sums (square metres) into
// hectares, rounded to 2 dp.
func HistogramFromGroups(groups []map[string]any) map[string]float64 {
	h := EmptyHistogram()
	for _, g := range groups {
		idx, ok := numberOf(g["bucket"])
		if !ok {
			idx = 0
		}
		sum, _ := numberOf(g["sum"])
		key := strconv.Itoa(int(idx))
		if _, valid := h[key]; valid {
			h[key] = Round2(sum / 10000.0)
		}
	}
	return h
}

func numberOf(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	}
	return 0, false
}

// ThresholdsFor returns the interpretation table for a band.
func ThresholdsFor(cfg *config.Config, band string) []config.Threshold {
	switch band {
	case "CVI":
		return cfg.CVIThresholds
	case "NDVI":
		return cfg.NDVIThresholds
	case "EVI":
		return cfg.EVIThresholds
	case "SAVI":
		return cfg.SAVIThresholds
	case "NDMI":
		return cfg.NDMIThresholds
	case "NDWI":
		return cfg.NDWIThresholds
	case "GNDVI":
		return cfg.GNDVIThresholds
	}
	return nil
}

// Interpretation strings from stats_service. The em dash is U+2014.
const (
	InterpNoData  = "N/A — data unavailable"
	InterpUnknown = "Unknown"
)

// BuildFarmSummary assembles farm_summary from already-fetched values.
//
// On ANY failure of the reduction steps the Python sets every mean to None,
// cvi_std to 0.0 and the histogram to all-zero — and STILL computes a
// confidence score from the scene count and cloud percentage. Callers reproduce
// that by passing nil means and a zero std (§7.8 step 6).
func BuildFarmSummary(
	cfg *config.Config,
	sceneCount int,
	avgCloudPct float64,
	cviStd float64,
	means map[string]*float64,
	histogram map[string]float64,
) FarmSummary {
	if histogram == nil {
		histogram = EmptyHistogram()
	}
	out := FarmSummary{
		Confidence:    ConfidenceScore(cfg, sceneCount, avgCloudPct, cviStd),
		SceneCount:    sceneCount,
		Indices:       make(map[string]IndexSummary, len(StatsBands)),
		NDVIHistogram: histogram,
	}
	for _, band := range StatsBands {
		mean := means[band]
		var rounded *float64
		if mean != nil {
			r := Round4(*mean)
			rounded = &r
		}
		out.Indices[band] = IndexSummary{
			Mean:           rounded,
			Interpretation: config.Interpret(mean, ThresholdsFor(cfg, band), InterpNoData, InterpUnknown),
		}
	}
	return out
}

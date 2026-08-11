package pipeline

import (
	"github.com/SanTiwari07/NDVI_satellite/internal/config"
	"github.com/SanTiwari07/NDVI_satellite/internal/gee/eeexpr"
)

// SMI = (VV − VV_dry) / (VV_wet − VV_dry), clamped to [0, 1]  (G19).
//
// With the configured bounds that divisor is 12.0. Wetter soil returns a
// stronger (less negative) VV, so a higher SMI means wetter soil. The bounds
// are fixed rather than per-scene so the index is comparable across dates.
func SMI(cfg *config.Config, vvDB eeexpr.Image) eeexpr.Image {
	span := cfg.SMIVVWetDB - cfg.SMIVVDryDB
	return vvDB.
		SubtractNum(cfg.SMIVVDryDB).
		DivideNum(span).
		Clamp(0.0, 1.0).
		Rename("SMI")
}

// RVI = 4·VH_lin / (VV_lin + VH_lin), clamped to [0, 1]  (G20).
//
// Backscatter must be converted from dB to linear power first:
// linear = 10^(dB/10), which serialises as Image.constant(10).pow(dB ÷ 10).
func RVI(vvDB, vhDB eeexpr.Image) eeexpr.Image {
	vvLin := eeexpr.ImageConstant(10.0).Pow(vvDB.DivideNum(10.0))
	vhLin := eeexpr.ImageConstant(10.0).Pow(vhDB.DivideNum(10.0))
	return vhLin.
		MultiplyNum(4.0).
		Divide(vvLin.Add(vhLin)).
		Clamp(0.0, 1.0).
		Rename("RVI")
}

// RATIO = VV_dB − VH_dB  (G21). Subtraction in dB is division in linear power.
func RATIO(vvDB, vhDB eeexpr.Image) eeexpr.Image {
	return vvDB.Subtract(vhDB).Rename("RATIO")
}

// ComputeRadarIndices produces the five-band radar image.
//
// Band order is select(["VV","VH"]) then addBands([SMI, RVI, RATIO]) — and, as
// with the vegetation path, the Python list argument becomes ONE addBands whose
// srcImg is the three indices already combined.
func ComputeRadarIndices(cfg *config.Config, composite eeexpr.Image) eeexpr.Image {
	vv := composite.Select("VV")
	vh := composite.Select("VH")

	combined := SMI(cfg, vv).
		AddBands(RVI(vv, vh)).
		AddBands(RATIO(vv, vh))

	return composite.Select("VV", "VH").AddBands(combined)
}

// ReduceRadarGridExpr reduces the five radar bands per grid cell (G23).
func ReduceRadarGridExpr(
	cfg *config.Config,
	b *eeexpr.Builder,
	img eeexpr.Image,
	grid eeexpr.Collection,
) eeexpr.Collection {
	return ReduceGridExpr(cfg, b, img, grid, cfg.RadarBands)
}

// RadarFarmMeanExpr is the farm-wide mean over the five radar bands (G24).
func RadarFarmMeanExpr(cfg *config.Config, img eeexpr.Image, geom eeexpr.Geometry) eeexpr.Node {
	return img.Select(cfg.RadarBands...).ReduceRegion(eeexpr.ReducerMean(), geom.N, 10, 1e9)
}

// ClassifyMoisture maps an SMI value to Dry / Moderate / Wet.
// A nil value yields "No data" — note that this differs from the vegetation
// path's "No data available".
func ClassifyMoisture(cfg *config.Config, smi *float64) string {
	return config.Interpret(smi, cfg.SMIThresholds, "No data", "Dry")
}

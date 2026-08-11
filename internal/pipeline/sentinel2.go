// Package pipeline builds the Earth Engine expression graphs for the analytics
// core and post-processes the results.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §7.
//
// Layering rule (§3.2): this package must never import net/http or Gin. It
// deals in expression graphs and plain data; internal/httpapi never builds
// graphs. That discipline is what makes the parity tests possible.
package pipeline

import (
	"time"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
	"github.com/SanTiwari07/NDVI_satellite/internal/gee/eeexpr"
)

// DateWindow is a [start, end) pair in YYYY-MM-DD form.
//
// Earth Engine's filterDate is END-EXCLUSIVE, and the Python code passes
// today as the end, so today's imagery is deliberately excluded (§7.1).
type DateWindow struct {
	Start string
	End   string
}

// LookbackWindow reproduces get_sentinel_composite's date maths.
//
// datetime.date.today() uses the SERVER'S LOCAL timezone, so deployed in IST
// the window rolls over at 00:00 IST. time.Now() is local here on purpose —
// using UTC would shift the window and change which scenes are selected (§13.8).
func LookbackWindow(cfg *config.Config, now time.Time) DateWindow {
	end := now
	start := end.AddDate(0, 0, -cfg.LookbackDays)
	return DateWindow{Start: start.Format("2006-01-02"), End: end.Format("2006-01-02")}
}

// SingleDayWindow reproduces get_single_day_composite: [target, target+1).
//
// The Python writes `target - timedelta(days=0)` for the start, which is a
// no-op; the resulting window is reproduced without the dead arithmetic (§7.2).
func SingleDayWindow(target string) (DateWindow, error) {
	t, err := time.Parse("2006-01-02", target)
	if err != nil {
		return DateWindow{}, err
	}
	return DateWindow{
		Start: t.Format("2006-01-02"),
		End:   t.AddDate(0, 0, 1).Format("2006-01-02"),
	}, nil
}

// MaskCloudsSCL builds the per-pixel cloud/shadow mask (G4).
//
// Image.constant(1), then .And(scl.neq(v)) for each masked class IN ORDER,
// then updateMask. The ordering is load-bearing: it determines the shape of the
// nested Image.and chain and therefore whether the graph matches the fixture.
func MaskCloudsSCL(cfg *config.Config, img eeexpr.Image) eeexpr.Image {
	scl := img.Select(cfg.Bands["SCL"])
	mask := eeexpr.ImageConstant(1)
	for _, class := range cfg.SCLMaskValues {
		mask = mask.And(scl.NeqNum(float64(class)))
	}
	return img.UpdateMask(mask)
}

// S2CollectionExpr builds the filtered, masked, scaled collection (G1).
//
// Order: filterBounds → filterDate → cloud filter → SCL mask → ÷10000.
//
// K4: the divide also scales the SCL band, which is meaningless but harmless
// because the mask has already been applied. Preserved for graph parity.
func S2CollectionExpr(cfg *config.Config, b *eeexpr.Builder, geom eeexpr.Geometry, w DateWindow) eeexpr.Collection {
	return eeexpr.ImageCollectionLoad(cfg.Dataset).
		FilterBounds(geom).
		FilterDate(w.Start, w.End).
		Filter(eeexpr.FilterLessThan("CLOUDY_PIXEL_PERCENTAGE", cfg.MaxCloudCoverPct)).
		Map(b, func(img eeexpr.Image) eeexpr.Image { return MaskCloudsSCL(cfg, img) }).
		Map(b, func(img eeexpr.Image) eeexpr.Image { return img.DivideNum(10000) })
}

// S2DatesCollectionExpr builds the collection behind /api/analyze-dates (G14).
//
// This is a DIFFERENT graph from S2CollectionExpr: bounds, date and cloud
// percentage only — no SCL mask and no ÷10000 scaling. Reusing the composite
// collection here would not match the running backend (§10.4).
func S2DatesCollectionExpr(cfg *config.Config, geom eeexpr.Geometry, w DateWindow) eeexpr.Collection {
	return eeexpr.ImageCollectionLoad(cfg.Dataset).
		FilterBounds(geom).
		FilterDate(w.Start, w.End).
		Filter(eeexpr.FilterLessThan("CLOUDY_PIXEL_PERCENTAGE", cfg.MaxCloudCoverPct))
}

// AvailableDatesExpr maps each scene to its acquisition date, then
// aggregate_array → distinct → sort (G14).
func AvailableDatesExpr(cfg *config.Config, b *eeexpr.Builder, geom eeexpr.Geometry, w DateWindow) eeexpr.Node {
	coll := S2DatesCollectionExpr(cfg, geom, w)
	dated := coll.MapToFeature(b, func(img eeexpr.Image) eeexpr.Node {
		millis := eeexpr.GetProperty(img.N, "system:time_start")
		return eeexpr.NewFeature(nil, eeexpr.Dict(map[string]eeexpr.Node{
			"date": eeexpr.DateFormat(millis, "YYYY-MM-dd"),
		})).N
	})
	return eeexpr.ListSort(eeexpr.ListDistinct(dated.AggregateArray("date")))
}

// SmoothTileExpr builds the bicubic-resampled tile image for a vegetation band
// (G13): select → clip → updateMask(gte 0) → resample → reproject → focal_mean.
//
// The updateMask(gte 0) step is vegetation-specific; the radar variant omits it
// because dB values are legitimately negative (G22).
func SmoothTileExpr(img eeexpr.Image, geom eeexpr.Geometry, band string) eeexpr.Image {
	return img.Select(band).
		Clip(geom).
		UpdateMask(img.Select(band).GteNum(0)).
		Resample("bicubic").
		Reproject("EPSG:4326", 10).
		FocalMean(2, "circle", "pixels")
}

// PointSampleExpr samples one pixel for the hover tooltip (G16).
// Reducer.first(), scale 10, maxPixels 1.
func PointSampleExpr(img eeexpr.Image, lng, lat float64, band string, scale float64) eeexpr.Node {
	return img.Select(band).ReduceRegion(
		eeexpr.ReducerFirst(),
		eeexpr.Point(lng, lat).N,
		scale,
		1,
	)
}

package pipeline

import (
	"time"

	"github.com/SanTiwari07/NDVI_satellite/internal/config"
	"github.com/SanTiwari07/NDVI_satellite/internal/gee/eeexpr"
)

// The Sentinel-1 radar pipeline is INDEPENDENT of the Sentinel-2 vegetation
// path (§7.9). It shares only the grid tiling and the Gaussian smoothing, both
// of which are geometry-only.
//
// S1 GRD VV/VH bands are already log-scaled (dB), so there is no ÷10000 step,
// and one orbit pass is pinned so backscatter is comparable across dates.

// S1CompositeWindow reproduces get_s1_composite's date maths (§7.9).
//
//	with a target date: [target − 6d, target + 7d)
//	  — the Python writes window_days + 1 so the exclusive end still includes
//	    target + 6d.
//	without one:        [today − 12d, today + 1d)
func S1CompositeWindow(cfg *config.Config, target string, now time.Time) (DateWindow, error) {
	if target != "" {
		centre, err := time.Parse("2006-01-02", target)
		if err != nil {
			return DateWindow{}, err
		}
		return DateWindow{
			Start: centre.AddDate(0, 0, -cfg.S1DateWindowDays).Format("2006-01-02"),
			End:   centre.AddDate(0, 0, cfg.S1DateWindowDays+1).Format("2006-01-02"),
		}, nil
	}
	return DateWindow{
		Start: now.AddDate(0, 0, -cfg.S1DateWindowDays*2).Format("2006-01-02"),
		End:   now.AddDate(0, 0, 1).Format("2006-01-02"),
	}, nil
}

// S1LookbackWindow is the window behind /api/analyze-radar-dates.
func S1LookbackWindow(cfg *config.Config, now time.Time) DateWindow {
	return DateWindow{
		Start: now.AddDate(0, 0, -cfg.S1LookbackDays).Format("2006-01-02"),
		End:   now.Format("2006-01-02"),
	}
}

// BaseS1CollectionExpr filters by mode, polarisation, resolution and orbit pass
// but NOT by date, then selects VV + VH (G17).
func BaseS1CollectionExpr(cfg *config.Config, b *eeexpr.Builder, geom eeexpr.Geometry) eeexpr.Collection {
	return eeexpr.ImageCollectionLoad(cfg.S1Dataset).
		FilterBounds(geom).
		Filter(eeexpr.FilterEquals("instrumentMode", cfg.S1InstrumentMode)).
		Filter(eeexpr.FilterListContains("transmitterReceiverPolarisation", "VV")).
		Filter(eeexpr.FilterListContains("transmitterReceiverPolarisation", "VH")).
		Filter(eeexpr.FilterEquals("resolution_meters", cfg.S1ResolutionM)).
		Filter(eeexpr.FilterEquals("orbitProperties_pass", cfg.S1OrbitPass)).
		Select(b, "VV", "VH")
}

// SpeckleFilter applies a focal-median smooth to the dB bands, then restores
// the band names and the acquisition timestamp (G18).
//
// copyProperties is required: median() and the date listing both depend on
// system:time_start surviving the map.
func SpeckleFilter(cfg *config.Config, img eeexpr.Image) eeexpr.Image {
	return img.
		FocalMedian(float64(cfg.S1SpeckleRadiusM), cfg.S1SpeckleKernel, "meters").
		Rename("VV", "VH").
		CopyProperties(img, []string{"system:time_start"})
}

// S1CollectionExpr is the speckle-filtered, date-restricted collection (G18).
func S1CollectionExpr(cfg *config.Config, b *eeexpr.Builder, geom eeexpr.Geometry, w DateWindow) eeexpr.Collection {
	return BaseS1CollectionExpr(cfg, b, geom).
		FilterDate(w.Start, w.End).
		Map(b, func(img eeexpr.Image) eeexpr.Image { return SpeckleFilter(cfg, img) })
}

// S1AvailableDatesExpr lists distinct S1 acquisition dates. Unlike the S2
// equivalent it uses the base collection unmodified — no speckle filter.
func S1AvailableDatesExpr(cfg *config.Config, b *eeexpr.Builder, geom eeexpr.Geometry, w DateWindow) eeexpr.Node {
	dated := BaseS1CollectionExpr(cfg, b, geom).
		FilterDate(w.Start, w.End).
		MapToFeature(b, func(img eeexpr.Image) eeexpr.Node {
			millis := eeexpr.GetProperty(img.N, "system:time_start")
			return eeexpr.NewFeature(nil, eeexpr.Dict(map[string]eeexpr.Node{
				"date": eeexpr.DateFormat(millis, "YYYY-MM-dd"),
			})).N
		})
	return eeexpr.ListSort(eeexpr.ListDistinct(dated.AggregateArray("date")))
}

// SmoothRadarTileExpr is G13 WITHOUT the updateMask(gte 0) step: radar dB
// values are legitimately negative, and masking them would blank the layer (G22).
func SmoothRadarTileExpr(img eeexpr.Image, geom eeexpr.Geometry, band string) eeexpr.Image {
	return img.Select(band).
		Clip(geom).
		Resample("bicubic").
		Reproject("EPSG:4326", 10).
		FocalMean(2, "circle", "pixels")
}

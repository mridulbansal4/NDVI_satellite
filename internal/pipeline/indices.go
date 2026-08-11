package pipeline

import (
	"github.com/SanTiwari07/NDVI_satellite/internal/config"
	"github.com/SanTiwari07/NDVI_satellite/internal/gee/eeexpr"
)

// IndexBands is the band order compute_all_indices produces, and the order
// every downstream select() depends on (§7.3).
var IndexBands = []string{"NDVI", "EVI", "SAVI", "NDMI", "NDWI", "GNDVI", "CVI"}

// StatsBands is stats_service's select order, which differs from IndexBands:
// CVI comes FIRST. The reduceRegion result is keyed by band name so the order
// does not change the numbers, but it does change the graph.
var StatsBands = []string{"CVI", "NDVI", "EVI", "SAVI", "NDMI", "NDWI", "GNDVI"}

// NDVI is (NIR − RED)/(NIR + RED). It, NDMI, NDWI and GNDVI are all
// normalizedDifference pairs (G5).
func NDVI(cfg *config.Config, img eeexpr.Image) eeexpr.Image {
	return img.NormalizedDifference(cfg.Bands["NIR"], cfg.Bands["RED"]).Rename("NDVI")
}

func NDMI(cfg *config.Config, img eeexpr.Image) eeexpr.Image {
	return img.NormalizedDifference(cfg.Bands["NIR"], cfg.Bands["SWIR"]).Rename("NDMI")
}

func NDWI(cfg *config.Config, img eeexpr.Image) eeexpr.Image {
	return img.NormalizedDifference(cfg.Bands["GREEN"], cfg.Bands["NIR"]).Rename("NDWI")
}

func GNDVI(cfg *config.Config, img eeexpr.Image) eeexpr.Image {
	return img.NormalizedDifference(cfg.Bands["NIR"], cfg.Bands["GREEN"]).Rename("GNDVI")
}

// EVI = 2.5·(NIR − RED) / (NIR + 6·RED − 7.5·BLUE + 1)
//
// PRD §7.3 / G6: the Python uses image.expression(...), which the serialiser
// turns into an Image.parseExpression invocation wrapping a synthesised custom
// function. Reproducing parseExpression in Go is unnecessary risk, so this is
// written as explicit band arithmetic — mathematically identical, and gated by
// the g06_evi_arith fixture plus a numeric equivalence check.
func EVI(cfg *config.Config, img eeexpr.Image) eeexpr.Image {
	nir := img.Select(cfg.Bands["NIR"])
	red := img.Select(cfg.Bands["RED"])
	blue := img.Select(cfg.Bands["BLUE"])
	return nir.Subtract(red).
		MultiplyNum(2.5).
		Divide(nir.Add(red.MultiplyNum(6.0)).Subtract(blue.MultiplyNum(7.5)).AddNum(1.0)).
		Rename("EVI")
}

// SAVI = ((NIR − RED) / (NIR + RED + 0.5)) · 1.5. Same rewrite rationale as EVI.
func SAVI(cfg *config.Config, img eeexpr.Image) eeexpr.Image {
	nir := img.Select(cfg.Bands["NIR"])
	red := img.Select(cfg.Bands["RED"])
	return nir.Subtract(red).
		Divide(nir.Add(red).AddNum(0.5)).
		MultiplyNum(1.5).
		Rename("SAVI")
}

// CVI is the weighted fusion, computed AFTER the six index bands are stacked
// and then added as a seventh band (§7.3).
//
// Weight order follows config.py's dict literal, because the chained
// .add() calls it produces are what the fixture records. NDWI carries no
// weight — that is not an omission.
var cviOrder = []string{"NDVI", "EVI", "SAVI", "NDMI", "GNDVI"}

func CVI(cfg *config.Config, stacked eeexpr.Image) eeexpr.Image {
	var acc eeexpr.Image
	for i, band := range cviOrder {
		term := stacked.Select(band).MultiplyNum(cfg.CVIWeights[band])
		if i == 0 {
			acc = term
			continue
		}
		acc = acc.Add(term)
	}
	return acc.Rename("CVI")
}

// ComputeAllIndices stacks the six indices onto the composite and appends CVI
// (G7), producing: original S2 bands, then NDVI/EVI/SAVI/NDMI/NDWI/GNDVI, then
// CVI. That order is what every downstream select() depends on.
//
// Nesting detail that matters: the Python writes
// composite.addBands([ndvi, …, gndvi]) with a LIST. The client turns the list
// into a single combined image first — chaining addBands among the six — and
// then performs ONE addBands against the composite. So the recursion is on the
// srcImg side, not the dstImg side. Chaining
// composite.addBands(ndvi).addBands(evi)… instead produces a different graph
// (verified against fixture g07_indexed_arith), even though the resulting band
// list is identical.
func ComputeAllIndices(cfg *config.Config, composite eeexpr.Image) eeexpr.Image {
	combined := NDVI(cfg, composite).
		AddBands(EVI(cfg, composite)).
		AddBands(SAVI(cfg, composite)).
		AddBands(NDMI(cfg, composite)).
		AddBands(NDWI(cfg, composite)).
		AddBands(GNDVI(cfg, composite))
	stacked := composite.AddBands(combined)
	return stacked.AddBands(CVI(cfg, stacked))
}

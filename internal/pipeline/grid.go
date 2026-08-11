package pipeline

import (
	"github.com/SanTiwari07/NDVI_satellite/internal/config"
	"github.com/SanTiwari07/NDVI_satellite/internal/gee/eeexpr"
)

// GridExpr tiles the polygon at a metre-based scale (G8).
//
// Projection('EPSG:4326').atScale(n) is what makes `n` mean metres rather than
// the projection's native degrees — that is the whole reason grid_service uses
// atScale instead of passing a scale to coveringGrid directly.
func GridExpr(cfg *config.Config, geom eeexpr.Geometry, scaleM int) eeexpr.Collection {
	proj := eeexpr.AtScale(eeexpr.Projection("EPSG:4326"), float64(scaleM))
	return geom.CoveringGrid(proj)
}

// ReduceGridExpr sets each cell's mean band values onto the cell feature (G9).
//
// reduceRegion(mean, cell.geometry(), scale=10, maxPixels=1e8), then
// cell.set(stats). One table:computeFeatures call materialises the whole grid.
func ReduceGridExpr(
	cfg *config.Config,
	b *eeexpr.Builder,
	img eeexpr.Image,
	grid eeexpr.Collection,
	bands []string,
) eeexpr.Collection {
	subset := img.Select(bands...)
	return grid.MapFeatures(b, func(cell eeexpr.Feature) eeexpr.Feature {
		stats := subset.ReduceRegion(
			eeexpr.ReducerMean(),
			cell.Geometry().N,
			float64(cfg.GridScaleM),
			1e8,
		)
		return cell.SetDict(stats)
	})
}

// GridScaleSearch reproduces generate_grid's auto-coarsening loop (§7.4).
//
// The loop performs one network round-trip per iteration and is a latency hot
// spot. PRD §7.4 permits estimating the initial scale locally ONLY if a test
// proves the final (scale, cellCount) pair is identical to Python's for at
// least five fixture polygons of varying size. That test does not exist, so the
// algorithm is kept exactly as-is and the round-trips are paid.
//
// sizeOf performs the value:compute call for a given scale.
func GridScaleSearch(cfg *config.Config, sizeOf func(scale int) (int, error)) (scale, cells int, err error) {
	scale = cfg.GridScaleM
	cells, err = sizeOf(scale)
	if err != nil {
		return 0, 0, err
	}
	for cells > cfg.MaxGridCells {
		scale += cfg.GridScaleStepM
		cells, err = sizeOf(scale)
		if err != nil {
			return 0, 0, err
		}
	}
	return scale, cells, nil
}

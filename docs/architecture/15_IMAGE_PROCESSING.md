# 15 — Image Processing & Resampling Engine

## Overview
To provide smooth, pixelation-free visualizations, Earth Engine rasters undergo spatial processing and resampling in `backend/services/gee_service.py` and `backend/services/grid_service.py`.

## Bicubic Resampling & Smoothing (`get_smooth_tile_url`)

```python
smooth_image = (
    image.select(band)
    .clip(ee_geometry)
    .updateMask(image.select(band).gte(0))
    .resample('bicubic')
    .reproject(crs='EPSG:4326', scale=10)
    .focal_mean(2, 'circle', 'pixels')
)
```

### Steps:
1. **Clip**: Restrict raster computation strictly to the polygon boundary.
2. **Mask Negatives**: Filter invalid values (`< 0`).
3. **Resample**: Apply `bicubic` interpolation to eliminate coarse square pixels.
4. **Reproject**: Reproject to WGS-84 (`EPSG:4326`) at native 10m resolution (`scale=10`).
5. **Focal Mean**: Apply 2-pixel circular focal smoothing kernel.

## Dynamic Grid Tiling & Auto-Coarsening (`grid_service.py`)
- **Base Grid Scale**: `GRID_SCALE_M = 10` (10 meters matching Sentinel-2 native resolution).
- **Cell Budget**: `MAX_GRID_CELLS = 2000`.
- **Auto-Scale Algorithm**: If polygon size yields $> 2000$ cells, `generate_grid()` increments scale by `GRID_SCALE_STEP_M` (2 meters per step) until cell count falls under budget.

## Related Documents
- [03_GOOGLE_EARTH_ENGINE.md](./03_GOOGLE_EARTH_ENGINE.md)
- [04_SENTINEL2_PIPELINE.md](./04_SENTINEL2_PIPELINE.md)
- [17_CONFIGURATION.md](./17_CONFIGURATION.md)

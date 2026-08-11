# 04 — Sentinel-2 Optical Pipeline

## Pipeline Execution Overview

The optical processing pipeline extracts Surface Reflectance from Sentinel-2 images to compute multi-spectral indices over a designated polygon geometry.

```
Raw GeoJSON Geometry
       |
       v
utils/geo_utils.py: geojson_to_ee_geometry()
       |
       v
services/gee_service.py: get_sentinel_composite()
       |
       +--> ee.ImageCollection("COPERNICUS/S2_SR_HARMONIZED")
       +--> .filterBounds(ee_geometry)
       +--> .filterDate(start_date, end_date) [LOOKBACK_DAYS = 90]
       +--> .filter(ee.Filter.lt("CLOUDY_PIXEL_PERCENTAGE", 20))
       +--> .map(_mask_clouds_scl)
       +--> .map(divide(10000))  # Scale DN to Reflectance [0.0, 1.0]
       |
       v
   collection.median()  --> ee.Image Composite
       |
       v
services/index_service.py: compute_all_indices(composite)
       |
       v
Multi-Band Image with: NDVI, EVI, SAVI, NDMI, NDWI, GNDVI, CVI
```

## SCL Cloud Masking Algorithm

The per-pixel Scene Classification Layer (`SCL`) masking function `_mask_clouds_scl(image)` filters out unreliable pixel classes:
- **Class 3**: Cloud Shadow
- **Class 8**: Medium Cloud Probability
- **Class 9**: High Cloud Probability
- **Class 10**: Thin Cirrus

Pixels matching any of these classes are masked out individually before median composite reduction.

## Single-Day Processing (`get_single_day_composite`)
When executing `/api/analyze-day`, the date window filters to `target_date` ± 1 day to account for orbital timing while retaining cloud masking.

## Related Documents
- [03_GOOGLE_EARTH_ENGINE.md](./03_GOOGLE_EARTH_ENGINE.md)
- [06_VEGETATION_INDICES.md](./06_VEGETATION_INDICES.md)
- [15_IMAGE_PROCESSING.md](./15_IMAGE_PROCESSING.md)

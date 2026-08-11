# 05 — Sentinel-1 SAR Radar Pipeline

## Pipeline Execution Overview

The Sentinel-1 Synthetic Aperture Radar (SAR) pipeline operates independently of the optical pipeline, providing cloud-penetrating radar intelligence via `backend/services/sar_service.py`.

```
GeoJSON Geometry + Date (Optional)
       |
       v
services/sar_service.py: get_s1_composite()
       |
       +--> ee.ImageCollection("COPERNICUS/S1_GRD")
       +--> .filterBounds(ee_geometry)
       +--> .filter(instrumentMode == "IW")
       +--> .filter(orbitProperties_pass == "DESCENDING")
       +--> .filterDate(start_date, end_date)
       +--> .select(["VV", "VH"])
       +--> .map(_apply_speckle_filter)  # Focal Median 50m Kernel
       |
       v
  collection.median() --> ee.Image Composite (dB backscatter)
       |
       v
services/radar_index_service.py: compute_radar_indices(composite)
       |
       v
Multi-Band SAR Image with: VV, VH, SMI, RVI, Ratio
```

## Speckle Noise Filtering
Radar images contain speckle noise. `sar_service.py` applies a spatial focal median filter:
- **Radius**: `S1_SPECKLE_RADIUS_M` (50 meters)
- **Kernel Shape**: Circle

## Calibration & Boundaries
- **Soil Moisture Index (SMI)** bounds (configured in `config.py`):
  - `SMI_VV_DRY_DB` = -20.0 dB (Dry soil reference)
  - `SMI_VV_WET_DB` = -8.0 dB (Wet soil reference)

## Related Documents
- [03_GOOGLE_EARTH_ENGINE.md](./03_GOOGLE_EARTH_ENGINE.md)
- [06_VEGETATION_INDICES.md](./06_VEGETATION_INDICES.md)
- [14_LAYER_SYSTEM.md](./14_LAYER_SYSTEM.md)

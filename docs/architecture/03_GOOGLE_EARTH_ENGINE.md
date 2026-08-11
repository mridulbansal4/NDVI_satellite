# 03 — Google Earth Engine Integration

## Overview
Google Earth Engine (GEE) serves as the primary cloud computation layer for processing satellite imagery in MindstriX. All GEE operations are orchestrated in `backend/services/gee_service.py` and `backend/services/sar_service.py`.

## Authentication & Initialization Flow

```
app.py Startup / Request
       |
       v
_init_gee_and_firebase_once()
       |
       v
initialize_gee() (services/gee_service.py)
       |
       +---> Check GEE_PROJECT_ID in config.py / .env
       |     If missing -> Log error, return False (GEE disabled)
       |
       +---> Check ~/.config/earthengine/credentials
       |     If missing -> Launch ee.Authenticate()
       |
       +---> ee.Initialize(project=GEE_PROJECT_ID)
       |
       v
  Connectivity Test: ee.Number(1).getInfo()
```

## Key API Call Rules
- **Isolation**: Only `gee_service.py` and `sar_service.py` call `ee.*` APIs directly.
- **Lazy Evaluation**: Band math, spatial clipping, and image collection filtering are lazily constructed on Google servers. Computation only executes when calling `.getInfo()` (for scene count or pixel sampling) or `.getMapId()` (for smooth tile URL generation).

## Related Documents
- [04_SENTINEL2_PIPELINE.md](./04_SENTINEL2_PIPELINE.md)
- [05_SENTINEL1_PIPELINE.md](./05_SENTINEL1_PIPELINE.md)
- [17_CONFIGURATION.md](./17_CONFIGURATION.md)

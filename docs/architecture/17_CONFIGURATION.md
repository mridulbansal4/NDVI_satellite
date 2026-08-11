# 17 — Configuration Parameters & Environment Variables

## Centralized Configuration (`backend/config.py`)

All tuning parameters, GEE datasets, and mathematical weights are configured in `backend/config.py`.

```python
# GEE Settings
GEE_PROJECT_ID      = os.getenv("GEE_PROJECT_ID", "")
DATASET             = "COPERNICUS/S2_SR_HARMONIZED"
LOOKBACK_DAYS       = 90
MAX_CLOUD_COVER_PCT = 20
SCL_MASK_VALUES     = [3, 8, 9, 10]  # Cloud shadow, medium cloud, high cloud, cirrus

# Composite Vegetation Index (CVI) Weights
CVI_WEIGHTS = {
    "NDVI":  0.70,
    "EVI":   0.10,
    "SAVI":  0.05,
    "NDMI":  0.10,
    "GNDVI": 0.05,
}

# Grid Tiling Parameters
GRID_SCALE_M      = 10
MAX_GRID_CELLS    = 2000
GRID_SCALE_STEP_M = 2

# Sentinel-1 SAR Settings
S1_DATASET          = "COPERNICUS/S1_GRD"
S1_INSTRUMENT_MODE  = "IW"
S1_ORBIT_PASS       = "DESCENDING"
S1_LOOKBACK_DAYS    = 90
S1_DATE_WINDOW_DAYS = 6
S1_SPECKLE_RADIUS_M = 50
SMI_VV_DRY_DB       = -20.0
SMI_VV_WET_DB       = -8.0

# Confidence Score Tuning
CONFIDENCE_SCENE_TARGET = 5
CONFIDENCE_STD_MAX      = 0.3
```

## Environment Variables (.env)

### `backend/.env`:
- `GEE_PROJECT_ID`: GCP Project ID with Earth Engine API enabled.
- `DATABASE_URL`: PostgreSQL connection string (`postgresql://user:pass@localhost:5432/mindstrix`).
- `FIREBASE_SERVICE_ACCOUNT_PATH`: Path to Firebase service account JSON.
- `JWT_SECRET_KEY`: Secret key for JWT token signing.
- `FLASK_PORT`: Server port (default 5000).

### `backend/chatbot/.env`:
- `OLLAMA_BASE_URL`: `http://localhost:11434`
- `OLLAMA_MODEL`: `llama3.2`

### `frontend/.env`:
- `VITE_API_BASE_URL`: Base API URL (empty to use Vite proxy in dev).

## Related Documents
- [06_VEGETATION_INDICES.md](./06_VEGETATION_INDICES.md)
- [08_BACKEND_ARCHITECTURE.md](./08_BACKEND_ARCHITECTURE.md)
- [HOW_TO_RUN.md](../HOW_TO_RUN.md)

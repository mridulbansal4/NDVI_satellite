# 02 — System Overview

## End-to-End Execution Sequence

```
+--------+       +------------+       +-----------+       +-------------+       +--------------+
| User   | ----> | Draw Farm  | ----> | POST API  | ----> | GEE Median  | ----> | Render Grid  |
| Client |       | Polygon    |       | /analyze  |       | Composite   |       | & Heatmap    |
+--------+       +------------+       +-----------+       +-------------+       +--------------+
    |                                                                                  |
    |                                                                                  v
    +--------------------------------------------------------------------> Ask Krishi Mitra
                                                                           Grounds LLM System
                                                                           Prompt with Stats
```

## Step-by-Step Workflow

1. **User Auth & Onboarding**: Farmer logs in via Firebase Phone OTP (`/api/auth/send-otp`, `/api/auth/verify-otp`). Upon authorization, the user completes the 9-step profile and farm registration sequence.
2. **Polygon Delineation**: On the interactive Leaflet map (`MapView.jsx`), the user draws or edits a farm boundary polygon (`GEOMETRY(POLYGON, 4326)`).
3. **Analysis Request**: The frontend issues a POST request to `/api/analyze` (Optical) or `/api/analyze-radar` (Radar) with the GeoJSON polygon object.
4. **Google Earth Engine Pipeline Execution**:
   - Backend converts GeoJSON to `ee.Geometry` (`utils/geo_utils.py`).
   - `gee_service.py` filters Sentinel-2 collection over the 90-day lookback window, applies SCL cloud/shadow masking, and reduces images to a median composite.
   - `index_service.py` computes all vegetation bands (NDVI, EVI, SAVI, NDMI, NDWI, GNDVI, CVI).
   - `grid_service.py` breaks the polygon into 10m grid cells, sampling values and applying Gaussian smoothing.
5. **Response & Visualization**: Backend returns a GeoJSON FeatureCollection along with farm-wide statistical metrics and GEE smooth map tile URLs. The React client renders the heatmap polygons (`HeatmapLayer.jsx`) and statistical sidebar (`FarmSummary.jsx`).
6. **Grounded AI Interaction**: The user asks questions in the Krishi Mitra chat panel (`KrishiMitraPanel.jsx`). The client sends the current `farmData` stats alongside the user query to `/chatbot/chat`, where the system prompt is dynamically assembled and evaluated by Ollama.

## Related Documents
- [ARCHITECTURE.md](./ARCHITECTURE.md)
- [04_SENTINEL2_PIPELINE.md](./architecture/04_SENTINEL2_PIPELINE.md)
- [07_API_ARCHITECTURE.md](./architecture/07_API_ARCHITECTURE.md)
- [12_CHATBOT.md](./architecture/12_CHATBOT.md)

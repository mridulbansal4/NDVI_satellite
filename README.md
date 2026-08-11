# Vegetation Index (VI) Engine 🛰️🌿

The **Vegetation Index Engine** is a production-grade, Google Earth Engine (GEE) powered module designed for the Satellite Agronomy Intelligence Platform. It goes beyond basic NDVI calculations by fusing multiple vegetation indices to provide a robust, highly accurate **Composite Vegetation Index (CVI)**. 

This tool is designed to convert raw Sentinel-2 satellite imagery into reliable, actionable farm-level insights, even in challenging atmospheric conditions or dense canopies.

---

## 🌟 Key Features

- **Composite Vegetation Index (CVI):** Fuses 6 distinct indices (NDVI, EVI, SAVI, NDMI, NDWI, GNDVI) linearly to overcome NDVI saturation and correctly assess vegetation health.
- **Advanced Preprocessing:** 
  - Uses Sentinel-2 Surface Reflectance (SR) data (not Top-Of-Atmosphere).
  - Performs per-pixel cloud and shadow masking using the Scene Classification Layer (SCL) band limit.
  - Generates robust median composites over sliding temporal windows.
- **Spatial Precision & Edge Correction:** Applies a 250m buffer and a 10m inward negative buffer to avoid edge mixed-pixels during statistical extraction.
- **Confidence Scoring:** Calculates an intelligent confidence metric (0-100%) based on scene availability, residual cloud cover, and spatial variance.
- **Time-Series Analysis:** Generates sliding-window CVI time series smoothed with a 3-point moving average to monitor plant health over seasons.
- **Modular & Configurable:** Clean architecture with a single tuning surface (`internal/config/config.go`) for thresholds, weights, datasets, palettes, and regions.

---

## 🧬 Why CVI over NDVI?

Basic NDVI is notorious for saturating in dense canopies (LAI > 3) and is highly sensitive to atmospheric noise and soil brightness. Our Composite Vegetation Index (CVI) mitigates this by fusing multiple indices:

| Weight | Index | Purpose |
| ------ | ----- | ------- |
| **0.70** | **NDVI** | The primary greenness signal, and highly robust against terrain shadows. |
| **0.10** | **EVI** | Corrects atmospheric noise and canopy saturation. |
| **0.05** | **SAVI** | Minimises soil background influence in sparse areas. |
| **0.10** | **NDMI** | Injects critical moisture metrics to detect drought stress. |
| **0.05** | **GNDVI** | Enhances sensitivity to chlorophyll and crop nutrient status. |

> These are the weights the engine actually uses, from
> `internal/config/config.go`. NDWI is computed and returned but carries **no**
> CVI weight. Earlier revisions of this table listed 0.35/0.25/0.15/0.15/0.10,
> which never matched the code — that discrepancy is K2 in
> [docs/KNOWN_ISSUES.md](docs/KNOWN_ISSUES.md) and is corrected here.

---

## 🛠️ Installation & Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/harshadd-ops/VI_engine.git
   cd VI_engine
   ```

2. **Install the Go toolchain** (1.23 or newer) and build:
   ```bash
   go build -o bin/server ./cmd/server
   ```

3. **Google Earth Engine authentication:**
   Production uses a GCP **service account registered for Earth Engine**. Point
   `GEE_SERVICE_ACCOUNT_KEY` at its JSON key. For local development you can
   instead reuse credentials from a previous `earthengine authenticate` run by
   setting `GEE_OAUTH_CLIENT_ID` and `GEE_OAUTH_CLIENT_SECRET`.
   Either way the GCP project needs the **Earth Engine API** enabled.

4. **Configure:**
   Copy `.env.example` to `.env` and set at least `GEE_PROJECT_ID`. The server
   starts even with no credentials at all — `/health` reports
   `gee_ready: false` and the analysis routes answer 503, while auth,
   onboarding and the dashboard keep working.

---

## 🚀 Usage

```bash
./bin/server            # listens on FLASK_PORT (default 5000)
curl http://127.0.0.1:5000/health
```

Draw a polygon in the frontend, or POST one directly:

```bash
curl -X POST http://127.0.0.1:5000/api/analyze   -H 'Content-Type: application/json'   -d '{"geometry": {"type":"Polygon","coordinates":[[[73.79,20.011],[73.7916,20.011],[73.7916,20.0122],[73.79,20.0122],[73.79,20.011]]]}}'
```

The response is a GeoJSON `FeatureCollection` of grid cells carrying NDVI, EVI,
SAVI, NDMI, NDWI, GNDVI and CVI, plus a `farm_summary` (confidence, scene count,
per-index means with interpretations, and a 20-bucket NDVI area histogram) and
bicubic-resampled tile URLs for each layer.

Every endpoint, with real verified responses:
[docs/ENDPOINT_VERIFICATION.md](docs/ENDPOINT_VERIFICATION.md).

---

## 📂 Architecture

The backend is Go. It was ported from Python/Flask under
`PRAGYA_GO_MIGRATION_PRD.md`; the Python implementation was removed at tag
`v2.0.0-go` and is recoverable with
`git checkout v2.0.0-go~1 -- legacy-python`.

- **`cmd/server`** — entrypoint. Wiring only; startup probes run once in
  goroutines and never block startup.
- **`internal/gee`** — the only package that talks to Earth Engine. There is no
  Go EE SDK, so `eeexpr` builds the expression-graph JSON by hand and posts it
  to the EE REST API.
- **`internal/pipeline`** — the analytics core: Sentinel-2 and Sentinel-1
  acquisition, the seven indices, grid tiling with auto-coarsening, Gaussian
  smoothing, statistics, and tile generation. Never touches HTTP.
- **`internal/httpapi`** — routing, validation and error envelopes. Never builds
  expression graphs.
- **`internal/config`** — the single tuning surface: datasets, bands, weights,
  thresholds, grid parameters, palettes.
- **`internal/repo` / `internal/db`** — hand-written SQL over pgx, preserving the
  PostGIS functions and `DISTINCT ON` / `ON CONFLICT` queries.
- **`internal/service`** — onboarding, SMS OTP, PIN lookup, dashboard assembly.
- **`internal/chatbot` / `internal/ollama`** — the Krishi Mitra assistant.
- **`testdata/`, `internal/*/testdata/`** — golden fixtures captured from the
  original Python implementation. They are what every test asserts against.

See [CLAUDE.md](CLAUDE.md) for the working guide and
[docs/KNOWN_ISSUES.md](docs/KNOWN_ISSUES.md) for behaviours that are
deliberately preserved.

---

## 🎯 Target Audience
Agronomic Data Scientists, Precision Agriculture Engineers, GIS developers, and Agritech startups building intelligent farm insight layers.

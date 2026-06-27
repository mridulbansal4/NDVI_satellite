# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Satellite Agronomy Intelligence Platform ("MindstriX"). Users draw farm-field polygons on a Leaflet map; the backend pulls Sentinel-2 imagery from Google Earth Engine (GEE), computes vegetation indices, and returns a smoothed per-cell heatmap grid plus farm statistics. A LangChain/Ollama chatbot ("Krishi Mitra") answers questions grounded in the current field's stats.

## Commands

This machine uses `py` (not `python`) and the venv lives at `backend/venv`.

```powershell
# One-time backend setup
cd C:\Projects\NDVI_satellite\backend
py -m venv venv
.\venv\Scripts\pip.exe install -r requirements.txt -r chatbot\requirements.txt
.\venv\Scripts\earthengine.exe authenticate   # opens browser for GEE OAuth

# Run backend (Flask, port 5000)
.\venv\Scripts\python.exe app.py
# Health check: GET http://127.0.0.1:5000/health → {"gee_ready": true, ...}

# Run frontend (Vite dev server, port 5173)
cd C:\Projects\NDVI_satellite\frontend
npm install        # first time only
npm run dev
npm run build      # production build → frontend/dist (Firebase hosting target)
npm run lint       # ESLint
```

There is no automated test suite. The GEE pipeline is validated by hitting the live endpoints with a polygon.

## Architecture

### Backend (`backend/`) — Flask + Google Earth Engine

`app.py` is the Flask entrypoint. It is a **pure REST API** (no HTML) with CORS open only to the Vite dev origins. GEE and Firebase are initialized lazily, exactly once, in a `@app.before_request` hook (`app._gee_ready` / `app._firebase_ready` flags).

Request flow for the core `/api/analyze` endpoint, with each stage owned by one service module — this layering is the key thing to understand:

1. `utils/geo_utils.py` — validate the GeoJSON polygon and convert it to an `ee.Geometry`.
2. `services/gee_service.py` — **the only module that talks to GEE directly.** Filters the Sentinel-2 collection (`COPERNICUS/S2_SR_HARMONIZED`) by bounds/date/cloud-cover, applies per-pixel SCL cloud+shadow masking, scales DN→reflectance (÷10000), and reduces to a median composite. Also produces bicubic-resampled tile URLs and single-pixel hover samples.
3. `services/index_service.py` — computes NDVI, EVI, SAVI, NDMI, NDWI, GNDVI on the composite, then a weighted **Composite Vegetation Index (CVI)** = weighted sum of those bands. Returns one multi-band `ee.Image`.
4. `services/grid_service.py` — tiles the polygon into a metre-based grid (`coveringGrid().atScale()`), auto-coarsening the scale to stay under `MAX_GRID_CELLS`, reduces each cell's index values, applies Gaussian spatial smoothing, attaches interpretation labels, and emits a GeoJSON FeatureCollection.
5. `services/stats_service.py` — computes the farm-wide summary + a 0–100 confidence score.

Everything operates lazily on GEE servers until `.getInfo()` / `.getMapId()` is called. The most recent indexed image + geometry are cached on the `app` object (`app._last_indexed_image`) so the `/api/sample` hover endpoint can sample without re-running the pipeline.

**Endpoints:** `/api/analyze` (median composite over last 90 days), `/api/analyze-dates` (list available S2 acquisition dates), `/api/analyze-day` (single-date NDVI), `/api/sample` (hover pixel value), `/api/auth/verify-token` (Firebase JWT verify), `/health`.

**`config.py` is the single tuning surface.** All GEE settings, band aliases, `CVI_WEIGHTS` (must sum to 1.0), grid resolution, cloud thresholds, and the index→interpretation threshold tables live here. Business logic reads from it — change behavior here, not in the service modules. Note: `config.py` weights/thresholds and the values quoted in `README.md` have drifted apart; trust `config.py`.

### Chatbot (`backend/chatbot/`) — Flask Blueprint + LangChain/Ollama

Registered as a blueprint under `/chatbot`. Layered so HTTP, prompt, and LLM concerns stay separate:
- `routes.py` — HTTP only (`/chat`, `/reset`, `/health`). Receives `farmData` + `heatmapData` from the frontend on every message.
- `prompts/` — `build_system_prompt(farm_data, heatmap_data)` injects the **current field's live stats** into the system prompt, so the prompt is rebuilt per request.
- `memory.py` — in-process per-`session_id` chat history (capped by `CHATBOT_MAX_HISTORY`).
- `chain.py` — builds the `ChatOllama` chain. The LLM is a **local Ollama server** (`OLLAMA_BASE_URL` / `OLLAMA_MODEL`); it must be running (`ollama serve`) with the model pulled, or `/chat` returns 502.

### Frontend (`frontend/`) — React 19 + Vite + Leaflet

- `App.jsx` — top-level state. Holds a **multi-field** model (`fields[]`, `activeFieldId`); each field carries its own geometry, analysis data, available dates, and selected date. An auth gate (`PremiumAuthFlow`) wraps the dashboard.
- `api.js` — the single backend client. All calls go same-origin and rely on Vite's proxy; set `VITE_API_BASE_URL` to target a non-proxied backend.
- `MapView.jsx` / `HeatmapLayer.jsx` — Leaflet map, polygon draw/edit (leaflet-draw, @turf/turf), and heatmap rendering of the returned grid.
- `firebase.js` + `PremiumAuthFlow.jsx` / `AuthModal.jsx` — Firebase phone-auth client. The client gets a JWT and the backend verifies it via Firebase Admin.

`vite.config.js` proxies `/api` (300s timeout — GEE calls are slow) and `/chatbot` to `http://127.0.0.1:5000`.

### Data / external services

- **GEE**: requires `GEE_PROJECT_ID` in `backend/.env` and stored OAuth creds (`earthengine authenticate`). Without a project ID the server logs an actionable error and `gee_ready` stays false.
- **Firebase**: `serviceAccountKey.json` (project root, gitignored) enables Firebase Admin. It's absent in dev, so `firebase_ready: false` is normal and only disables auth features.
- **PostgreSQL + PostGIS**: schema in `mindstrix_setup.sql`, setup walkthrough in `DATABASE_SETUP.md`. (Relational data store; not yet wired into the Flask request path.)

### `backend/legacy/`

The original interactive CLI version of the engine (`main.py`, `gee_engine.py`). Superseded by the `services/` + `app.py` web architecture — reference only, not imported by the running app.

## Conventions

- Config over code: tune `backend/config.py` and `.env`; avoid hardcoding thresholds/weights in service modules.
- Keep GEE calls confined to `gee_service.py`; other modules pass `ee.Image` / `ee.Geometry` objects around.
- `.env` files (both `backend/` and `frontend/`) and `serviceAccountKey.json` are gitignored — never commit secrets.

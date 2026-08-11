# 07 — API Architecture & Endpoints

## Overview
The backend provides a REST API using Flask 3. Global CORS is configured in `app.py` for Vite frontend environments.

## API Endpoint Matrix

| Method | Endpoint | Description | Authentication | Blueprint |
|---|---|---|---|---|
| `GET` | `/health` | Server & GEE health status | None | Core `app.py` |
| `POST` | `/api/analyze` | Sentinel-2 median analysis (90-day window) | Optional / Public | Core `app.py` |
| `POST` | `/api/analyze-dates` | Get available S2 scene acquisition dates | Public | Core `app.py` |
| `POST` | `/api/analyze-day` | Analyze single-day Sentinel-2 scene | Public | Core `app.py` |
| `GET` | `/api/sample` | Single pixel value sampling (hover) | Public | Core `app.py` |
| `POST` | `/api/analyze-radar` | Sentinel-1 SAR moisture analysis | Public | Core `app.py` |
| `POST` | `/api/analyze-radar-dates` | Get available S1 scene acquisition dates | Public | Core `app.py` |
| `POST` | `/api/auth/send-otp` | Request Firebase SMS OTP | Public | `blueprints/auth.py` |
| `POST` | `/api/auth/verify-otp` | Verify SMS OTP | Public | `blueprints/auth.py` |
| `POST` | `/api/auth/verify-token` | Verify Firebase Client JWT | Public | Core `app.py` |
| `POST` | `/chatbot/chat` | Send query to Krishi Mitra LLM | Public | `chatbot/routes.py` |
| `POST` | `/chatbot/reset` | Clear chatbot session history | Public | `chatbot/routes.py` |
| `GET` | `/chatbot/health` | Check Ollama LLM availability | Public | `chatbot/routes.py` |

## Response Formats

### `POST /api/analyze` Response Schema:
```json
{
  "type": "FeatureCollection",
  "features": [ ... ],
  "farm_summary": {
    "confidence": 85.5,
    "scene_count": 4,
    "ndvi": 0.72,
    "cvi": 0.68,
    "period_start": "2026-04-28",
    "period_end": "2026-07-27"
  },
  "farm_boundary": { ... },
  "ndvi_tile_url": "https://earthengine.googleapis.com/v1/...",
  "tile_url": "https://earthengine.googleapis.com/v1/...",
  "index_tiles": {
    "ndvi_tile_url": "...",
    "evi_tile_url": "...",
    "cvi_tile_url": "..."
  }
}
```

## Related Documents
- [08_BACKEND_ARCHITECTURE.md](./08_BACKEND_ARCHITECTURE.md)
- [10_FIREBASE_AUTH.md](./10_FIREBASE_AUTH.md)
- [12_CHATBOT.md](./12_CHATBOT.md)

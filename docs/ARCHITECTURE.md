# High-Level Architecture

The MindstriX platform is constructed as a decoupled, multi-tier web application combining a React SPA frontend, a Flask REST API backend, Google Earth Engine processing pipelines, a dual-database architecture (PostgreSQL + PostGIS & Cloud Firestore), and an offline LLM service.

```
+-----------------------------------------------------------------------+
|                            Browser Client                             |
|    React 19 + Vite SPA (Port 5173) | Leaflet Map + Heatmap Grids      |
+-----------------------------------+-----------------------------------+
                                    | HTTP REST / JSON (JWT Auth)
                                    v
+-----------------------------------------------------------------------+
|                            Flask REST API                             |
|    app.py (Port 5000) | Blueprints (Auth, Farm, Farmer, Chatbot)   |
+---------+-------------------------+-----------------------+-----------+
          |                         |                       |
          v                         v                       v
+-------------------+     +--------------------+  +---------------------+
| Google Earth Engine|     | PostgreSQL + PostGIS|  |    Cloud Firestore  |
| S2 Optical & S1   |     | Relational Schema  |  |  Sessions & Alerts  |
| SAR Pipelines     |     | Farms & Reports    |  |  Real-time pushes   |
+-------------------+     +--------------------+  +---------------------+
          ^
          | Context Prompting
+---------+----------+
|  Ollama + LangChain|
|  Krishi Mitra LLM  |
+--------------------+
```

## Core Subsystems
1. **Frontend**: React 19 SPA using Leaflet and `leaflet-draw` for interactive polygon drawing, map rendering, and statistical visualizations.
2. **Backend**: Python 3 Flask server using Blueprints for modular route routing, `psycopg2` connection pooling, and GEE Python SDK wrappers.
3. **Data Storage**:
   - **PostgreSQL 16 + PostGIS 3**: Primary relational store for spatial farm boundaries (`GEOMETRY(POLYGON, 4326)`), farmer profiles, crops, and VI reports.
   - **Cloud Firestore**: Real-time store for ephemeral onboarding sessions (`farmer_sessions`), OTP verification (`otp_sessions`), and live notifications (`farm_alerts`).
4. **Processing Pipeline**: Google Earth Engine (Sentinel-2 SR & Sentinel-1 GRD) lazily queried for median composites and converted to custom GeoJSON grids (`grid_service.py`).
5. **AI Chatbot**: Ollama hosting `llama3.2` orchestrated via LangChain (`backend/chatbot`), receiving grounded farm metrics dynamically.

## Related Documents
- [02_SYSTEM_OVERVIEW.md](./02_SYSTEM_OVERVIEW.md)
- [07_API_ARCHITECTURE.md](./architecture/07_API_ARCHITECTURE.md)
- [08_BACKEND_ARCHITECTURE.md](./architecture/08_BACKEND_ARCHITECTURE.md)
- [09_FRONTEND_ARCHITECTURE.md](./architecture/09_FRONTEND_ARCHITECTURE.md)
- [11_DATABASE.md](./architecture/11_DATABASE.md)

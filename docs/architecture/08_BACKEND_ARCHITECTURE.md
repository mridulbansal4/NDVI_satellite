# 08 — Backend Architecture

## Directory Layout (`backend/`)

```
backend/
├── app.py                      # Flask REST API entry point
├── config.py                   # Global configuration & constants
├── db/                         # PostgreSQL connection pool (pool.py)
├── blueprints/                 # Flask Modular Blueprints
│   ├── auth.py                 # OTP send/verify routes
│   ├── farmer.py               # Farmer profile routes
│   ├── farm.py                 # Farm polygon routes
│   ├── crop.py                 # Crop data routes
│   ├── irrigation.py           # Irrigation info routes
│   ├── soil.py                 # Soil data routes
│   ├── consent.py              # Consent logging & triggering
│   └── dashboard.py            # Dashboard aggregate routes
├── services/                   # Business Logic & GEE wrappers
│   ├── gee_service.py          # Sentinel-2 GEE pipeline
│   ├── index_service.py        # Optical index formulas & CVI
│   ├── grid_service.py         # GeoJSON grid generation & smoothing
│   ├── stats_service.py        # Statistics calculation
│   ├── sar_service.py          # Sentinel-1 GEE pipeline
│   ├── radar_index_service.py  # Radar formulas (SMI, RVI, Ratio)
│   ├── radar_grid_service.py   # Radar grid reduction
│   ├── auth_service.py         # Firebase Admin SDK initialization & JWT
│   └── sms_service.py          # Fast2SMS OTP integration
├── repositories/               # PostgreSQL Database Access Layer
├── firestore/                  # Firestore session sync
└── chatbot/                    # Ollama + LangChain chatbot module
```

## Architectural Design Patterns
1. **Blueprint Modularization**: Routing is strictly divided into domain-specific blueprints (`blueprints/`).
2. **Service Layer Isolation**: GEE calls, index calculations, and database access are encapsulated into service modules (`services/`). `app.py` only orchestrates handlers.
3. **Lazy Initialization**: Firebase and GEE services are lazily initialized via `@app.before_request` in `app.py`. Failures do not prevent server launch.

## Related Documents
- [07_API_ARCHITECTURE.md](./07_API_ARCHITECTURE.md)
- [11_DATABASE.md](./11_DATABASE.md)
- [17_CONFIGURATION.md](./17_CONFIGURATION.md)

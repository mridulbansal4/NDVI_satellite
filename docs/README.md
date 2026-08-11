# MindstriX Technical Documentation Suite

Welcome to the comprehensive, enterprise-grade engineering documentation suite for the **MindstriX Satellite Agronomy Intelligence Platform**.

This documentation is maintained directly against the codebase with zero documentation drift.

---

## Document Index

### Root Documentation
- [**PROJECT_CONTEXT.md**](./PROJECT_CONTEXT.md): Mission context, agriculture domain, Sentinel-1/Sentinel-2 integration.
- [**ARCHITECTURE.md**](./ARCHITECTURE.md): System-wide architecture, component interactions, and data flow.
- [**01_INTRODUCTION.md**](./01_INTRODUCTION.md): Platform goals, core capabilities, and target audience.
- [**02_SYSTEM_OVERVIEW.md**](./02_SYSTEM_OVERVIEW.md): End-to-end user and technical execution lifecycle.
- [**CHANGELOG.md**](./CHANGELOG.md): Historical record of codebase and documentation changes.
- [**CONTRIBUTING.md**](./CONTRIBUTING.md): Engineering standards, code style, and contribution workflows.
- [**CURRENT_STATE.md**](./CURRENT_STATE.md): Operational status of all platform features.
- [**HOW_TO_RUN.md**](./HOW_TO_RUN.md): Local environment setup, dependencies, and execution guide.
- [**VALIDATION.md**](./VALIDATION.md): Verification procedures and endpoint validation.
- [**DOCUMENTATION_AUDIT_REPORT.md**](./DOCUMENTATION_AUDIT_REPORT.md): Audit report verifying zero documentation drift against source files.

### Subsystem Architecture (`docs/architecture/`)
- [**03_GOOGLE_EARTH_ENGINE.md**](./architecture/03_GOOGLE_EARTH_ENGINE.md): GEE SDK initialization, credentials, and API connection management.
- [**04_SENTINEL2_PIPELINE.md**](./architecture/04_SENTINEL2_PIPELINE.md): Optical imagery acquisition, cloud masking (SCL), median composite math.
- [**05_SENTINEL1_PIPELINE.md**](./architecture/05_SENTINEL1_PIPELINE.md): Synthetic Aperture Radar (SAR) processing, IW mode, speckle filtering.
- [**06_VEGETATION_INDICES.md**](./architecture/06_VEGETATION_INDICES.md): Spectral index math (NDVI, EVI, SAVI, NDMI, NDWI, GNDVI, CVI, SMI, RVI, Ratio).
- [**07_API_ARCHITECTURE.md**](./architecture/07_API_ARCHITECTURE.md): REST API specification, endpoint parameter schemas, JSON responses.
- [**08_BACKEND_ARCHITECTURE.md**](./architecture/08_BACKEND_ARCHITECTURE.md): Flask structure, blueprint layout, repository pattern, service layer.
- [**09_FRONTEND_ARCHITECTURE.md**](./architecture/09_FRONTEND_ARCHITECTURE.md): React 19 SPA architecture, state management, Vite routing.
- [**10_FIREBASE_AUTH.md**](./architecture/10_FIREBASE_AUTH.md): Phone OTP authentication, Firebase SDK, JWT validation middleware.
- [**11_DATABASE.md**](./architecture/11_DATABASE.md): PostgreSQL + PostGIS spatial schema and Cloud Firestore real-time session stores.
- [**12_CHATBOT.md**](./architecture/12_CHATBOT.md): Krishi Mitra AI assistant, LangChain + ChatOllama, dynamic prompt context injection.
- [**13_MAP_RENDERING.md**](./architecture/13_MAP_RENDERING.md): Leaflet, Leaflet-Draw, and GeoJSON grid rendering.
- [**14_LAYER_SYSTEM.md**](./architecture/14_LAYER_SYSTEM.md): Multi-layer visualization, palette gradients, Optical vs. Radar mode toggle.
- [**15_IMAGE_PROCESSING.md**](./architecture/15_IMAGE_PROCESSING.md): Grid tiling, bicubic resampling, scale coarsening algorithms.
- [**16_ERROR_HANDLING.md**](./architecture/16_ERROR_HANDLING.md): Exception handling, fallback mechanisms, GEE failure handling.
- [**17_CONFIGURATION.md**](./architecture/17_CONFIGURATION.md): Centralized parameters (`backend/config.py`), environment variables.

### Dashboard & UI Subsystems (`docs/dashboard/`)
- [**18_DASHBOARD.md**](./dashboard/18_DASHBOARD.md): Main dashboard UI, multi-field state management.
- [**19_MAP_PAGE.md**](./dashboard/19_MAP_PAGE.md): Polygon creation, editing, boundary management, map view state.
- [**20_ANALYSIS_PAGE.md**](./dashboard/20_ANALYSIS_PAGE.md): Index visualization, acquisition timeline bar, statistical charts.
- [**21_AUTHENTICATION_UI.md**](./dashboard/21_AUTHENTICATION_UI.md): Premium authentication screens, OTP modals, user login flow.
- [**22_COMPONENTS.md**](./dashboard/22_COMPONENTS.md): Detailed inventory of all reusable React frontend components.

### System Evaluation (`docs/evaluation/`)
- [**23_PERFORMANCE.md**](./evaluation/23_PERFORMANCE.md): Latency analysis, GEE processing benchmarks, database connection pooling.
- [**24_LIMITATIONS.md**](./evaluation/24_LIMITATIONS.md): Single-thread blocking limits, memory constraints, polygon size thresholds.
- [**25_KNOWN_ISSUES.md**](./evaluation/25_KNOWN_ISSUES.md): Known edge cases, cloud cover scene availability, GEE token expiration.

### Future Development (`docs/future/`)
- [**26_FUTURE_WORK.md**](./future/26_FUTURE_WORK.md): Planned infrastructure upgrades, background task queues.
- [**27_ROADMAP.md**](./future/27_ROADMAP.md): Architectural roadmap and feature evolution goals.

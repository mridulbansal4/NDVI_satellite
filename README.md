<div align="center">

<img src="https://capsule-render.vercel.app/api?type=waving&color=0:0d1117,50:161b22,100:21262d&height=200&section=header&text=MindstriX&fontSize=68&fontColor=58a6ff&animation=fadeIn&fontAlignY=35&desc=Satellite+Farm+Analysis+Platform&descAlignY=58&descColor=8b949e&descSize=20" width="100%"/>

<img src="https://readme-typing-svg.demolab.com?font=JetBrains+Mono&size=15&duration=3000&pause=900&color=58A6FF&center=true&vCenter=true&multiline=true&width=760&height=55&lines=Sentinel-2+Optical+%7C+Sentinel-1+SAR+Radar+%7C+Real-time+Analysis;NDVI+%7C+EVI+%7C+SAVI+%7C+NDMI+%7C+NDWI+%7C+GNDVI+%7C+CVI+%7C+SMI+%7C+RVI;Krishi+Mitra+—+Grounded+Agronomy+AI+on+Live+Farm+Data" alt="Typing SVG"/>

<br/><br/>

<img src="https://img.shields.io/badge/Go-1.23%2B-00ADD8?style=flat-square&logo=go&logoColor=white"/>
<img src="https://img.shields.io/badge/Gin-1.10%2B-008ECF?style=flat-square&logo=go&logoColor=white"/>
<img src="https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react&logoColor=black"/>
<img src="https://img.shields.io/badge/Vite-8.0-646CFF?style=flat-square&logo=vite&logoColor=white"/>
<img src="https://img.shields.io/badge/Google_Earth_Engine-API-4285F4?style=flat-square&logo=google&logoColor=white"/>
<img src="https://img.shields.io/badge/Firebase-Auth_%26_Firestore-FFCA28?style=flat-square&logo=firebase&logoColor=black"/>
<img src="https://img.shields.io/badge/PostgreSQL_%2B_PostGIS-16-4169E1?style=flat-square&logo=postgresql&logoColor=white"/>
<img src="https://img.shields.io/badge/Gemini-2.5_Flash-4285F4?style=flat-square&logo=google&logoColor=white"/>

<br/><br/>

</div>

---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Architecture Overview](#architecture-overview)
- [Documentation](#documentation)
- [Installation](#installation)
- [Configuration](#configuration)
- [API Overview](#api-overview)
- [Satellite Analysis](#satellite-analysis)
- [Contributing](#contributing)

---

## Overview

### Purpose

MindstriX is a full-stack satellite agronomy intelligence platform that delivers high-resolution multi-spectral and radar satellite imagery analysis directly to farm-level workflows. The platform transforms raw **Sentinel-2** optical and **Sentinel-1 SAR** radar bands into actionable vegetation health heatmaps, soil moisture assessments, and AI-grounded agronomic recommendations.

### Problem Statement

Small and medium-scale farmers lack access to affordable, real-time satellite crop health monitoring. Traditional field scouting is slow, expensive, and incomplete. Existing cloud-based satellite platforms are complex and not optimized for the agronomist or farmer workflow.

### System Layers

| Layer | Technology | Responsibility |
|---|---|---|
| Interactive Map | React + Leaflet | Draw and submit farm boundary polygons |
| REST API | Go + Gin, Earth Engine REST API | Satellite imagery processing pipeline |
| Optical Pipeline | Sentinel-2 SR (COPERNICUS/S2_SR_HARMONIZED) | Cloud-free vegetation index heatmaps at 10m |
| Radar Pipeline | Sentinel-1 GRD (COPERNICUS/S1_GRD) | Cloud-penetrating soil moisture and vegetation maps |
| AI Assistant | Google Gemini (Ollama fallback), called directly over HTTP | Grounded agronomy Q&A on live farm data |
| Authentication | Firebase Phone OTP + JWT | Secure, mobile-first farmer login |
| Database | PostgreSQL + PostGIS | Persistent farmer, farm, and crop data |

---

## Features

All features listed are implemented and verified in the current codebase.

### Remote Sensing

| Feature | Implementation |
|---|---|
| Sentinel-2 90-day cloud-free median composite | `internal/gee/`, `internal/pipeline/sentinel2.go` |
| SCL-based per-pixel cloud and shadow masking | `internal/gee/`, `internal/pipeline/sentinel2.go` |
| Vegetation index computation: NDVI, EVI, SAVI, NDMI, NDWI, GNDVI, CVI | `internal/pipeline/indices.go` |
| Sentinel-1 SAR speckle-filtered VV/VH composite | `internal/pipeline/sentinel1.go` |
| Radar index computation: SMI, RVI, VV, VH, Ratio | `internal/pipeline/radar_indices.go` |
| 10m heatmap grid generation with Gaussian smoothing | `internal/pipeline/grid.go`, `radar_grid.go` |
| GEE bicubic-smoothed tile URL generation | `internal/gee/`, `internal/pipeline/sentinel2.go` |
| Single-date Sentinel-2 and Sentinel-1 analysis | `internal/httpapi/` |
| Single-pixel hover sampling | `internal/httpapi/analyze_handler.go` (`/api/sample`) |

### Frontend

| Feature | Implementation |
|---|---|
| Interactive Leaflet map with polygon drawing | `frontend/src/MapView.jsx`, `HeatmapLayer.jsx` |
| Farm boundary delineation and rendering | `frontend/src/MapView.jsx` |
| Optical and Radar layer mode toggle | `frontend/src/LayerToggle.jsx` |
| Vegetation index heatmap grid renderer | `frontend/src/HeatmapLayer.jsx` |
| Farm statistics sidebar | `frontend/src/FarmSummary.jsx`, `Legend.jsx` |
| Timeline bar for date navigation | `frontend/src/TimelineBar.jsx` |
| Krishi Mitra AI chatbot panel | `frontend/src/KrishiMitraPanel.jsx` |

### Authentication and Data

| Feature | Implementation |
|---|---|
| Firebase Phone OTP login | `internal/httpapi/onboarding_handler.go`, `internal/service/sms.go` |
| Firebase JWT token verification | `internal/firebase/admin.go` |
| 8-step farmer onboarding flow | `frontend/src/pages/steps/`, `internal/httpapi/` |
| PostgreSQL + PostGIS data persistence | `internal/db/`, `internal/repo/`, `schema.sql` |
| Firestore ephemeral session synchronisation | `internal/firestore/` |
| Protected routes and JWT session management | `frontend/src/App.jsx` |

---

## Tech Stack

### Frontend

<div align="center">

![React](https://img.shields.io/badge/React_19-20232A?style=flat-square&logo=react&logoColor=61DAFB)
![Vite](https://img.shields.io/badge/Vite_8-646CFF?style=flat-square&logo=vite&logoColor=white)
![React Router](https://img.shields.io/badge/React_Router_7-CA4245?style=flat-square&logo=react-router&logoColor=white)
![Tailwind](https://img.shields.io/badge/Tailwind_CSS_3-0F172A?style=flat-square&logo=tailwindcss&logoColor=38BDF8)
![Leaflet](https://img.shields.io/badge/Leaflet_1.9-199900?style=flat-square&logo=leaflet&logoColor=white)
![Recharts](https://img.shields.io/badge/Recharts_3-22C55E?style=flat-square)
![Axios](https://img.shields.io/badge/Axios_1-5A29E4?style=flat-square&logo=axios&logoColor=white)
![Turf.js](https://img.shields.io/badge/Turf.js_7-00A896?style=flat-square)
![Firebase SDK](https://img.shields.io/badge/Firebase_SDK_12-FFCA28?style=flat-square&logo=firebase&logoColor=black)

</div>

| Package | Version | Role |
|---|---|---|
| react | 19 | UI component framework |
| vite | 8 | Build tool and dev server |
| react-router-dom | 7 | Client-side routing |
| leaflet / react-leaflet | 1.9 / 5.0 | Interactive map rendering |
| leaflet-draw | 1.0 | Polygon drawing tools |
| recharts | 3 | Data visualisation charts |
| axios | 1 | HTTP API client |
| @turf/turf | 7 | GeoJSON geometry operations |
| tailwindcss | 3 | Utility-first styling |
| firebase | 12 | Firebase Auth client |

### Backend

<div align="center">

![Go](https://img.shields.io/badge/Go_1.23%2B-00ADD8?style=flat-square&logo=go&logoColor=white)
![Gin](https://img.shields.io/badge/Gin_1.10%2B-008ECF?style=flat-square&logo=go&logoColor=white)
![JWT](https://img.shields.io/badge/golang--jwt_v5-1F2937?style=flat-square)
![pgx](https://img.shields.io/badge/pgx_v5-4169E1?style=flat-square&logo=postgresql&logoColor=white)

</div>

| Package | Version | Role |
|---|---|---|
| gin-gonic/gin | 1.10+ | REST API framework |
| gin-contrib/cors | 1.7+ | Cross-origin request handling |
| golang-jwt/jwt | v5 | JWT issuance and validation, wire-compatible with the previous Flask-JWT-Extended tokens |
| go-playground/validator | v10 | Input validation and schema enforcement |
| jackc/pgx | v5 | PostgreSQL driver and connection pool |
| joho/godotenv | 1.5+ | Environment variable loading |
| golang.org/x/crypto | latest | scrypt / pbkdf2, for verifying existing Werkzeug password hashes |

### GIS and Remote Sensing

<div align="center">

![GEE](https://img.shields.io/badge/Google_Earth_Engine-REST_v1-4285F4?style=flat-square&logo=google&logoColor=white)
![Sentinel-2](https://img.shields.io/badge/Sentinel--2_SR_L2A-10m_Optical-2E7D32?style=flat-square)
![Sentinel-1](https://img.shields.io/badge/Sentinel--1_GRD-SAR_Radar-1565C0?style=flat-square)

</div>

| Technology | Role |
|---|---|
| Earth Engine REST API (`earthengine.googleapis.com/v1`) | Sentinel-2 and Sentinel-1 collection filtering, compositing, index computation, and tile generation. No Go EE SDK exists, so `internal/gee/eeexpr` builds the expression graphs by hand and posts them to `value:compute`, `table:computeFeatures` and `maps`. |
| COPERNICUS/S2_SR_HARMONIZED | Optical multispectral imagery — 10m/20m resolution |
| COPERNICUS/S1_GRD | C-band SAR radar imagery — VV/VH polarisation, DESCENDING orbit pass |

### Database and Authentication

<div align="center">

![PostgreSQL](https://img.shields.io/badge/PostgreSQL_16-4169E1?style=flat-square&logo=postgresql&logoColor=white)
![PostGIS](https://img.shields.io/badge/PostGIS_3.x-336791?style=flat-square)
![Firebase Auth](https://img.shields.io/badge/Firebase_Auth_Phone_OTP-FFCA28?style=flat-square&logo=firebase&logoColor=black)
![Firebase Admin](https://img.shields.io/badge/Firebase_Admin_SDK-FFA000?style=flat-square&logo=firebase&logoColor=black)

</div>

| Technology | Role |
|---|---|
| PostgreSQL 16 | Primary relational store for farmer, farm, crop, irrigation, and soil records |
| PostGIS 3.x | Spatial extension — farm polygons stored as GEOMETRY(POLYGON, 4326) |
| jackc/pgx v5 | PostgreSQL driver and connection pool (min 2 / max 20) |
| Firebase Auth (Phone) | Phone number OTP-based authentication |
| Firebase Admin SDK | Server-side Firebase JWT token verification |

### AI

<div align="center">

![Gemini](https://img.shields.io/badge/Gemini_2.5_Flash-Hosted_LLM-4285F4?style=flat-square&logo=google&logoColor=white)
![Ollama](https://img.shields.io/badge/Ollama_llama3.2-Local_fallback-00A67E?style=flat-square)
![Go](https://img.shields.io/badge/net%2Fhttp-Direct_Ollama_API-00ADD8?style=flat-square&logo=go&logoColor=white)

</div>

| Technology | Role |
|---|---|
| Google Gemini (`gemini-2.5-flash`) | Hosted LLM, selected by setting `GEMINI_API_KEY` |
| Ollama (llama3.2) | Local LLM runtime, used when no Gemini key is configured — no external API dependency |
| `internal/gemini`, `internal/ollama` | Direct HTTP clients, interchangeable behind one interface. LangChain was dropped in the Go port: the chain was a system prompt plus history plus one user turn, which is a short HTTP client. |
| `internal/chatbot` | Per-session conversation memory and the system-prompt template, rebuilt per request from the current field's live statistics |

---

## Project Structure

```
NDVI_satellite-1/
|
+-- frontend/                        # React + Vite web application
|   +-- src/
|   |   +-- pages/
|   |   |   +-- Welcome.jsx
|   |   |   +-- Login.jsx
|   |   |   +-- Signup.jsx
|   |   |   +-- Analysis.jsx         # Main satellite analysis page
|   |   |   +-- steps/               # 8-step onboarding (Step1 - Step8)
|   |   +-- components/
|   |   |   +-- DataPanel.jsx
|   |   |   +-- FarmCard.jsx
|   |   |   +-- InputField.jsx
|   |   |   +-- PrimaryButton.jsx
|   |   |   +-- ProgressBar.jsx
|   |   |   +-- SelectField.jsx
|   |   |   +-- VIReportBadge.jsx
|   |   +-- context/
|   |   |   +-- OnboardingContext.jsx
|   |   +-- api/                     # API client modules
|   |   +-- utils/
|   |   +-- styles/
|   |   +-- MapView.jsx              # Leaflet map with polygon drawing
|   |   +-- HeatmapLayer.jsx         # GeoJSON heatmap cell renderer
|   |   +-- KrishiMitraPanel.jsx     # AI chatbot sidebar panel
|   |   +-- FarmSummary.jsx          # Statistics sidebar
|   |   +-- Legend.jsx               # Colour scale legend
|   |   +-- TimelineBar.jsx          # Date selection timeline
|   |   +-- LayerToggle.jsx          # Optical / Radar mode toggle
|   |   +-- firebase.js              # Firebase client initialisation
|   |   +-- api.js                   # Axios API client
|   |   +-- colorUtils.js
|   |   +-- main.jsx
|   +-- package.json
|   +-- vite.config.js
|   +-- tailwind.config.js
|
+-- cmd/
|   +-- server/
|       +-- main.go                  # Entrypoint: config -> deps -> router
|
+-- internal/                        # Go backend (ported from Flask)
|   +-- config/                      # THE single tuning surface
|   +-- gee/                         # The only package that talks to Earth Engine
|   |   +-- session.go               # Service-account / developer OAuth
|   |   +-- client.go                # value:compute, computeFeatures, maps
|   |   +-- eeexpr/                  # Expression-graph builder + fixtures
|   +-- pipeline/                    # The analytics core (no HTTP here)
|   |   +-- sentinel2.go             # S2 composite, dates, single-day
|   |   +-- sentinel1.go             # S1 base collection, speckle filter
|   |   +-- indices.go               # NDVI/EVI/SAVI/NDMI/NDWI/GNDVI + CVI
|   |   +-- radar_indices.go         # SMI, RVI, RATIO
|   |   +-- grid.go, radar_grid.go   # Tiling and per-cell reduction
|   |   +-- smooth.go                # Gaussian spatial smoothing
|   |   +-- stats.go                 # Farm statistics and confidence
|   |   +-- cache.go                 # TTL'd expression cache
|   +-- httpapi/                     # Routing, validation, error envelopes
|   |   +-- middleware/              # CORS and JWT
|   +-- repo/                        # Hand-written SQL (no ORM)
|   +-- db/                          # pgxpool
|   +-- service/                     # Onboarding, SMS OTP, PIN lookup
|   +-- crypto/                      # Werkzeug password-hash compatibility
|   +-- jwtutil/                     # Flask-JWT-Extended-compatible tokens
|   +-- firebase/, firestore/        # ID-token verification, session sync
|   +-- chatbot/, ollama/            # Krishi Mitra
|   +-- geo/, logging/
|
+-- tools/
|   +-- contract/                    # HTTP contract runner (73 cases)
|
+-- testdata/                        # Golden fixtures + fixture polygons
|
+-- docs/                            # Technical documentation
|   +-- architecture/                # 03-17  architecture reference docs
|   +-- dashboard/                   # 18-22  frontend UI docs
|   +-- evaluation/                  # 23-25  performance and known issues
|   +-- future/                      # 26-27  roadmap and future work
|
+-- public/
+-- schema.sql                       # PostgreSQL production schema
+-- mindstrix_setup.sql
+-- migrate_add_password.sql
+-- firebase.json
+-- .firebaserc
+-- .gitignore
```

---

## Architecture Overview

```
+-------------------------------------------------------------+
|                      User (Browser)                         |
+-----------------------------+-------------------------------+
                              |
                              v
+-------------------------------------------------------------+
|          React + Vite Frontend  (port 5173)                 |
|  - Leaflet map  :  draw farm boundary polygon               |
|  - Submit GeoJSON geometry to backend REST API              |
|  - Render heatmap grid and GEE tile overlay                 |
|  - Statistics sidebar, chatbot panel, layer toggle          |
+-----------------------------+-------------------------------+
                              |
                              v
+-------------------------------------------------------------+
|          Go REST API Backend  (port 5000)                   |
|  - Validate GeoJSON polygon                                 |
|  - Convert to ee.Geometry via geo_utils.py                  |
|  - Route to optical or radar pipeline                       |
+----------+----------------------------------+---------------+
           |                                  |
           v                                  v
+---------------------+          +----------------------------+
| Sentinel-2 Optical  |          | Sentinel-1 Radar Pipeline  |
| gee_service.py      |          | sar_service.py             |
| - 90-day lookback   |          | - S1 GRD collection filter |
| - SCL cloud mask    |          | - Speckle filter (focal    |
| - Median composite  |          |   median, ~50m radius)     |
|          |          |          | - Median composite         |
|          v          |          |          |                  |
| index_service.py    |          | radar_index_service.py     |
| NDVI  EVI  SAVI     |          | SMI   RVI   VV/VH   Ratio  |
| NDMI  NDWI  GNDVI   |          |                            |
| CVI (weighted sum)  |          |                            |
+----------+----------+          +-------------+--------------+
           |                                   |
           +---------------+-------------------+
                           |
                           v
             grid_service / radar_grid_service
             - 10m grid cell generation
             - Per-cell band value reduction (ee.Reducer)
             - Gaussian smoothing
                           |
                           v
             Response: GeoJSON FeatureCollection
                      + GEE smooth tile URLs
                      + Farm-wide statistics
                           |
                           v
+-------------------------------------------------------------+
|               React Frontend — Visualisation                |
|  HeatmapLayer.jsx   ->  per-cell coloured polygons          |
|  GEE tile overlay   ->  bicubic-smoothed raster             |
|  FarmSummary.jsx    ->  vegetation index statistics         |
|  Legend.jsx         ->  colour gradient scale               |
+-----------------------------+-------------------------------+
                              |
                              v
              KrishiMitraPanel.jsx
              POST /chatbot/chat
              Body: { farmData, heatmapData, message }
                              |
                              v
              internal/chatbot  ->  grounded system prompt
              Ollama (llama3.2)  ->  agronomic answer
```

---

## Documentation

Detailed technical documentation is maintained in [`/docs`](./docs/). The README serves as the entry point; all architecture decisions, formulas, configuration references, and known limitations are documented in the files below.

### Core Documents

| Document | Description |
|---|---|
| [01_INTRODUCTION.md](./docs/01_INTRODUCTION.md) | Platform vision and core capabilities |
| [02_SYSTEM_OVERVIEW.md](./docs/02_SYSTEM_OVERVIEW.md) | End-to-end execution sequence |
| [ARCHITECTURE.md](./docs/ARCHITECTURE.md) | High-level system architecture |
| [CURRENT_STATE.md](./docs/CURRENT_STATE.md) | Subsystem implementation status matrix |
| [HOW_TO_RUN.md](./docs/HOW_TO_RUN.md) | Detailed setup and run instructions |
| [CONTRIBUTING.md](./docs/CONTRIBUTING.md) | Contribution guidelines and architectural rules |
| [VALIDATION.md](./docs/VALIDATION.md) | Validation methodology |
| [CHANGELOG.md](./docs/CHANGELOG.md) | Version history |
| [PROJECT_CONTEXT.md](./docs/PROJECT_CONTEXT.md) | Project background and context |

### Architecture Reference (`docs/architecture/`)

| Document | Description |
|---|---|
| [03_GOOGLE_EARTH_ENGINE.md](./docs/architecture/03_GOOGLE_EARTH_ENGINE.md) | GEE API integration |
| [04_SENTINEL2_PIPELINE.md](./docs/architecture/04_SENTINEL2_PIPELINE.md) | Sentinel-2 processing pipeline |
| [05_SENTINEL1_PIPELINE.md](./docs/architecture/05_SENTINEL1_PIPELINE.md) | Sentinel-1 SAR radar pipeline |
| [06_VEGETATION_INDICES.md](./docs/architecture/06_VEGETATION_INDICES.md) | Vegetation index formulas and CVI design |
| [07_API_ARCHITECTURE.md](./docs/architecture/07_API_ARCHITECTURE.md) | REST API routes and contracts |
| [08_BACKEND_ARCHITECTURE.md](./docs/architecture/08_BACKEND_ARCHITECTURE.md) | Backend service layer design |
| [09_FRONTEND_ARCHITECTURE.md](./docs/architecture/09_FRONTEND_ARCHITECTURE.md) | Frontend component architecture |
| [10_FIREBASE_AUTH.md](./docs/architecture/10_FIREBASE_AUTH.md) | Firebase Phone OTP authentication |
| [11_DATABASE.md](./docs/architecture/11_DATABASE.md) | PostgreSQL schema and PostGIS usage |
| [12_CHATBOT.md](./docs/architecture/12_CHATBOT.md) | Krishi Mitra chatbot architecture |
| [13_MAP_RENDERING.md](./docs/architecture/13_MAP_RENDERING.md) | Leaflet map and heatmap rendering |
| [14_LAYER_SYSTEM.md](./docs/architecture/14_LAYER_SYSTEM.md) | Optical vs. Radar layer toggle system |
| [15_IMAGE_PROCESSING.md](./docs/architecture/15_IMAGE_PROCESSING.md) | Image processing and tile generation |
| [16_ERROR_HANDLING.md](./docs/architecture/16_ERROR_HANDLING.md) | Error handling strategy |
| [17_CONFIGURATION.md](./docs/architecture/17_CONFIGURATION.md) | Centralised configuration reference |

### Dashboard and UI (`docs/dashboard/`)

| Document | Description |
|---|---|
| [18_DASHBOARD.md](./docs/dashboard/18_DASHBOARD.md) | Dashboard overview |
| [19_MAP_PAGE.md](./docs/dashboard/19_MAP_PAGE.md) | Map page |
| [20_ANALYSIS_PAGE.md](./docs/dashboard/20_ANALYSIS_PAGE.md) | Analysis page |
| [21_AUTHENTICATION_UI.md](./docs/dashboard/21_AUTHENTICATION_UI.md) | Authentication UI |
| [22_COMPONENTS.md](./docs/dashboard/22_COMPONENTS.md) | Reusable UI components |

### Evaluation and Roadmap

| Document | Description |
|---|---|
| [23_PERFORMANCE.md](./docs/evaluation/23_PERFORMANCE.md) | Performance characteristics |
| [24_LIMITATIONS.md](./docs/evaluation/24_LIMITATIONS.md) | Known limitations |
| [25_KNOWN_ISSUES.md](./docs/evaluation/25_KNOWN_ISSUES.md) | Known issues |
| [26_FUTURE_WORK.md](./docs/future/26_FUTURE_WORK.md) | Future work |
| [27_ROADMAP.md](./docs/future/27_ROADMAP.md) | Development roadmap |

---

## Installation

### Prerequisites

| Requirement | Minimum Version | Notes |
|---|---|---|
| Python | 3.10 | Backend runtime |
| Node.js | 20 | Frontend build toolchain |
| PostgreSQL | 16 | With PostGIS 3.x extension |
| Google Cloud project | — | Earth Engine API must be enabled |
| Firebase project | — | Phone Authentication must be enabled |
| Ollama | latest | `llama3.2` model must be pulled |

### 1. Clone the Repository

```bash
git clone https://github.com/SanTiwari07/NDVI_satellite.git
cd NDVI_satellite-1
```

### 2. Backend Setup

```powershell
# Build the backend (Go 1.23+)
go build -o bin/server ./cmd/server

# Configure environment variables
copy .env.example .env
# Edit .env and set: GEE_PROJECT_ID, DATABASE_URL, FIREBASE_PROJECT_ID, JWT_SECRET_KEY

# Google Earth Engine credentials
# Production: point GEE_SERVICE_ACCOUNT_KEY at a service-account JSON key that
#             has been registered for Earth Engine.
# Local dev:  reuse an existing `earthengine authenticate` refresh token by also
#             setting GEE_OAUTH_CLIENT_ID and GEE_OAUTH_CLIENT_SECRET.

# Start the API server
.in\server
```

The server starts even with no credentials at all: `/health` reports
`gee_ready: false`, the analysis routes answer 503, and auth, onboarding and the
dashboard keep working.

The backend API will be available at `http://127.0.0.1:5000`.

### 3. Frontend Setup

```powershell
cd frontend
npm install
npm run dev
```

The frontend dev server will be available at `http://localhost:5173`.

### 4. Database Setup

```sql
psql -U postgres
CREATE DATABASE mindstrix;
\c mindstrix
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
\i schema.sql
```

### 5. Ollama Chatbot Service

```bash
ollama serve
ollama pull llama3.2
```

---

## Configuration

### Backend — `.env`

Copy `.env.example` to `.env` at the repository root and populate all variables.

| Variable | Description |
|---|---|
| `GEE_PROJECT_ID` | Google Cloud project ID with Earth Engine API enabled |
| `DATABASE_URL` | PostgreSQL connection string: `postgresql://user:password@host:5432/dbname` |
| `FIREBASE_PROJECT_ID` | Firebase project ID for Admin SDK initialisation |
| `JWT_SECRET_KEY` | HS256 signing key. Tokens stay wire-compatible with the previous Flask-JWT-Extended ones, so existing sessions survive — change in production |
| `FLASK_PORT` | Server port (default: `5000`). The env var name is kept for ops continuity |
| `FLASK_ENV` | `development` or `production` |

### Frontend — `frontend/.env`

| Variable | Description |
|---|---|
| `VITE_FIREBASE_API_KEY` | Firebase web API key |
| `VITE_FIREBASE_AUTH_DOMAIN` | Firebase auth domain |
| `VITE_FIREBASE_PROJECT_ID` | Firebase project ID |
| `VITE_FIREBASE_STORAGE_BUCKET` | Firebase storage bucket |
| `VITE_FIREBASE_MESSAGING_SENDER_ID` | Firebase messaging sender ID |
| `VITE_FIREBASE_APP_ID` | Firebase app ID |

> GEE credentials are written to `~/.config/earthengine/credentials` by `earthengine authenticate`. No service account key file is required.

---

## API Overview

Full API contracts are documented in [`docs/architecture/07_API_ARCHITECTURE.md`](./docs/architecture/07_API_ARCHITECTURE.md).

### Satellite Analysis

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Health check — GEE and Firebase initialisation status |
| `POST` | `/api/analyze` | Sentinel-2 analysis: vegetation indices + heatmap grid + farm stats |
| `POST` | `/api/analyze-dates` | List available Sentinel-2 acquisition dates for a polygon |
| `POST` | `/api/analyze-day` | Single-date Sentinel-2 analysis |
| `GET` | `/api/sample` | Single-pixel hover sampling (`?lat=&lng=&band=NDVI`) |
| `POST` | `/api/analyze-radar` | Sentinel-1 radar analysis: SMI, RVI, VV/VH ratio + grid |
| `POST` | `/api/analyze-radar-dates` | List available Sentinel-1 acquisition dates for a polygon |

### Authentication

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/auth/send-otp` | Send a 6-digit OTP to an Indian mobile number |
| `POST` | `/api/auth/verify-otp` | Verify OTP and issue a JWT session token |
| `POST` | `/api/auth/verify-token` | Verify a Firebase ID token via Admin SDK |

### Chatbot

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/chatbot/chat` | Send a user message; receive a grounded AI reply |
| `POST` | `/chatbot/reset` | Clear the conversation memory for a session |
| `GET` | `/chatbot/health` | Chatbot health check — Ollama model name and base URL |

---

## Satellite Analysis

### Sentinel-2 — Vegetation Indices

Computed by `internal/pipeline/indices.go` from a scaled Sentinel-2 SR median composite with reflectance values in `[0.0, 1.0]`.

| Index | Formula | Agronomic Meaning |
|---|---|---|
| NDVI | `(NIR - RED) / (NIR + RED)` | Overall vegetation greenness and canopy density |
| EVI | `2.5 * (NIR - RED) / (NIR + 6*RED - 7.5*BLUE + 1)` | Atmospheric-corrected vegetation; reduces canopy saturation |
| SAVI | `((NIR - RED) / (NIR + RED + 0.5)) * 1.5` | Vegetation with soil brightness bias correction |
| NDMI | `(NIR - SWIR) / (NIR + SWIR)` | Canopy moisture content and drought stress |
| NDWI | `(GREEN - NIR) / (GREEN + NIR)` | Water body detection |
| GNDVI | `(NIR - GREEN) / (NIR + GREEN)` | Chlorophyll content and nutritional status |
| CVI | `0.70·NDVI + 0.10·EVI + 0.05·SAVI + 0.10·NDMI + 0.05·GNDVI` | Composite multi-index fusion. Weights live in `internal/config/config.go` and must sum to 1.0; NDWI is computed and returned but carries no weight |

### Sentinel-1 — Radar Indices

Computed by `internal/pipeline/radar_indices.go` from a speckle-filtered VV/VH dB composite.

| Index | Formula | Agronomic Meaning |
|---|---|---|
| VV | Raw backscatter (dB) | Surface soil moisture and roughness sensitivity |
| VH | Raw backscatter (dB) | Vegetation volume structure and canopy scattering |
| SMI | `(VV - VV_dry) / (VV_wet - VV_dry)`, clamped to [0, 1] | Soil Moisture Index: 0 = dry, 1 = wet |
| RVI | `4 * VH_linear / (VV_linear + VH_linear)`, clamped to [0, 1] | Radar Vegetation Index — canopy density from SAR backscatter |
| Ratio | `VV_dB - VH_dB` | VV/VH ratio: soil dominance vs. vegetation dominance |

The Sentinel-1 pipeline is fully independent of cloud cover conditions, enabling continuous monitoring regardless of weather.

---

## Contributing

Read [`docs/CONTRIBUTING.md`](./docs/CONTRIBUTING.md) before submitting any changes.

### Architectural Rules

| Rule | Requirement |
|---|---|
| GEE Isolation | All Earth Engine calls must remain strictly within `internal/gee`. `internal/pipeline` builds expression graphs but performs no I/O, and `internal/httpapi` builds no graphs at all. Other packages only pass `eeexpr.Image` / `eeexpr.Geometry` values around. |
| Centralised Configuration | All threshold values, index weights, cloud cover limits, and array bounds must be defined in `internal/config/config.go`. Do not scatter constants through service logic. |
| No Silent Fallbacks | Return the correct HTTP error code on failure: `503` for GEE unavailable, `400` for invalid polygon. Do not swallow errors with fake or empty fallback data. |
| Zero Documentation Drift | API route changes must be reflected in `docs/architecture/07_API_ARCHITECTURE.md`. Index formula changes must be reflected in both `index_service.py` and `docs/architecture/06_VEGETATION_INDICES.md`. |

---

<div align="center">

<img src="https://capsule-render.vercel.app/api?type=waving&color=0:21262d,50:161b22,100:0d1117&height=110&section=footer&animation=fadeIn" width="100%"/>

For detailed technical documentation, architecture decisions, configuration reference, and known limitations, refer to the [`/docs`](./docs/) directory.

</div>

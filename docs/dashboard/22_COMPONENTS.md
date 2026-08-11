# 22 — Frontend Component Inventory

## Complete Component Matrix (`frontend/src/components/`)

| Component | File Path | Primary Responsibility |
|---|---|---|
| **MapView** | `src/components/MapView.jsx` | Leaflet map instance, tile layers, and polygon drawing controls |
| **HeatmapLayer** | `src/components/HeatmapLayer.jsx` | Renders backend GeoJSON 10m grid polygons with dynamic opacity |
| **FarmSummary** | `src/components/FarmSummary.jsx` | Displays index statistics (NDVI, CVI, confidence) and Recharts graphs |
| **TimelineBar** | `src/components/TimelineBar.jsx` | Scrollable date picker for single-day Sentinel-2 scene evaluation |
| **LayerToggle** | `src/components/LayerToggle.jsx` | Switcher toggling between Optical (S2) and Radar (S1) map modes |
| **Legend** | `src/components/Legend.jsx` | Dynamic color scale legend for active index layer |
| **KrishiMitraPanel** | `src/components/KrishiMitraPanel.jsx` | Conversational slide-out panel for Krishi Mitra AI assistant |
| **LocationForm** | `src/components/LocationForm.jsx` | Address entry form with India Post PIN code resolution |
| **PremiumAuthFlow** | `src/components/PremiumAuthFlow.jsx` | Firebase OTP login modal manager |
| **AuthModal** | `src/components/AuthModal.jsx` | OTP verification input dialog |
| **LoadingOverlay** | `src/components/LoadingOverlay.jsx` | Full-screen spinner active during GEE analysis execution |

## Related Documents
- [09_FRONTEND_ARCHITECTURE.md](../architecture/09_FRONTEND_ARCHITECTURE.md)
- [18_DASHBOARD.md](./18_DASHBOARD.md)

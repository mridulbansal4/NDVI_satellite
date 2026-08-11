# 18 — Dashboard Subsystem & Layout

## Overview
The dashboard application (`frontend/src/pages/Analysis.jsx`) integrates map rendering, statistics display, date selection, layer switching, and AI chatbot drawer into a unified SPA workbench.

## Structural Layout Breakdown

```
+-------------------------------------------------------------------------------+
| Header / Navbar (Field Switcher, Auth State, Mode Toggle)                     |
+------------------------------------+------------------------------------------+
|                                    | Sidebar Panel (`FarmSummary.jsx`)        |
| Interactive Leaflet Map            | - CVI Score & Confidence Metric          |
| (`MapView.jsx` & `HeatmapLayer`)   | - NDVI, EVI, SAVI, NDMI, NDWI Charts     |
|                                    | - Layer Toggle (`LayerToggle.jsx`)       |
| Floating Map Controls:             | - Chat Trigger (`KrishiMitraPanel`)      |
| - Draw Controls (`leaflet-draw`)   +------------------------------------------+
| - Color Legend (`Legend.jsx`)      | Timeline Acquisition Picker              |
|                                    | (`TimelineBar.jsx`)                      |
+------------------------------------+------------------------------------------+
```

## State Lifecycle
1. User loads dashboard.
2. Selects or draws field boundary polygon.
3. System triggers POST request to backend `/api/analyze`.
4. Renders 10m grid overlay and populates stats sidebar.

## Related Documents
- [09_FRONTEND_ARCHITECTURE.md](../architecture/09_FRONTEND_ARCHITECTURE.md)
- [19_MAP_PAGE.md](./19_MAP_PAGE.md)
- [20_ANALYSIS_PAGE.md](./20_ANALYSIS_PAGE.md)

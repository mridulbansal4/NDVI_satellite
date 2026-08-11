# 20 — Analysis & Statistics Subsystem (`FarmSummary.jsx`)

## Overview
The analysis view is driven by `frontend/src/components/FarmSummary.jsx` and `frontend/src/components/TimelineBar.jsx`.

## Key Visualizations & Metrics
- **CVI Overall Health Metric**: Displays Composite Vegetation Index mean, median, and confidence score.
- **Index Charts**: Employs `recharts` to chart relative band values (NDVI, EVI, SAVI, NDMI, NDWI, GNDVI).
- **Timeline Acquisition Bar (`TimelineBar.jsx`)**: Displays available acquisition dates fetched from `/api/analyze-dates`. Selecting a date triggers single-day composite analysis via `/api/analyze-day`.

## Related Documents
- [06_VEGETATION_INDICES.md](../architecture/06_VEGETATION_INDICES.md)
- [18_DASHBOARD.md](./18_DASHBOARD.md)

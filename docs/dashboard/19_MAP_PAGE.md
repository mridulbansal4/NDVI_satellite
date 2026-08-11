# 19 — Map Subsystem (`MapView.jsx`)

## Map Page Implementation

The interactive map is implemented in `frontend/src/components/MapView.jsx` using `react-leaflet`.

## Core Features
1. **Base Layer Tiles**: Standard OpenStreetMap / Satellite base tiles.
2. **Polygon Delineation**: Uses `leaflet-draw` to let users sketch custom farm boundary polygons directly on the map.
3. **Coordinate Normalization**: Automatically converts Leaflet layer coordinates into GeoJSON WGS84 `[longitude, latitude]` structure.
4. **Hover Pixel Sampling**: Sends cursor latitude/longitude to `/api/sample?lat=...&lng=...&band=NDVI` on mouse hover to populate instantaneous pixel tooltips.

## Related Documents
- [13_MAP_RENDERING.md](../architecture/13_MAP_RENDERING.md)
- [18_DASHBOARD.md](./18_DASHBOARD.md)

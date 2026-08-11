# 13 — Map Rendering & Heatmap Subsystem

## Map Architecture (`frontend/src/components/`)

Interactive mapping relies on Leaflet (`leaflet`), `react-leaflet`, and `leaflet-draw`.

```
           +---------------------------------------+
           |           MapView.jsx                 |
           | Initializes MapContainer & TileLayer  |
           +-------------------+-------------------+
                               |
            +------------------+------------------+
            |                                     |
            v                                     v
+-----------------------+             +-----------------------+
|  FeatureGroup         |             |  HeatmapLayer.jsx     |
|  (leaflet-draw)       |             |  Renders GeoJSON      |
|  Polygon Creation     |             |  FeatureCollection    |
+-----------------------+             +-----------------------+
```

## Dynamic Polygon Drawing
- `MapView.jsx` attaches Leaflet Draw controls allowing the user to create, edit, or delete farm polygons.
- Upon polygon completion (`onCreated`), coordinates are normalized into a standard GeoJSON `Polygon` structure.

## Grid Heatmap Layer (`HeatmapLayer.jsx`)
- Receives GeoJSON FeatureCollection emitted by backend `/api/analyze`.
- Iterates over features, rendering each 10m grid cell polygon.
- Styles each cell dynamically using cell properties (`cvi`, `color`, `fillOpacity`).

## Related Documents
- [09_FRONTEND_ARCHITECTURE.md](./09_FRONTEND_ARCHITECTURE.md)
- [14_LAYER_SYSTEM.md](./14_LAYER_SYSTEM.md)
- [15_IMAGE_PROCESSING.md](./15_IMAGE_PROCESSING.md)

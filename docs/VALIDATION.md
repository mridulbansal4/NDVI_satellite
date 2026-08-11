# System Validation Procedures

## Empirical Validation Methodology

The MindstriX platform utilizes live Google Earth Engine data pipelines; verification relies on endpoint contract testing and runtime geometry assertions.

## 1. Backend Health Check Validation

Request:
`GET http://127.0.0.1:5000/health`

Expected Response (200 OK):
```json
{
  "status": "ok",
  "gee_ready": true,
  "firebase_ready": true,
  "project": "your-gee-project-id"
}
```

## 2. GeoJSON Polygon Analysis Validation

Request:
`POST http://127.0.0.1:5000/api/analyze`

Payload:
```json
{
  "geometry": {
    "type": "Polygon",
    "coordinates": [
      [
        [75.52, 13.41],
        [75.54, 13.41],
        [75.54, 13.43],
        [75.52, 13.43],
        [75.52, 13.41]
      ]
    ]
  }
}
```

Verification Criteria:
- HTTP 200 Response.
- Response contains `farm_summary` object with `confidence`, `scene_count`, `ndvi`, `evi`, `savi`, `ndmi`, `ndwi`, `gndvi`, `cvi` keys.
- Response contains `features` array containing cell GeoJSON features with 10m grid boundaries.
- Response contains `ndvi_tile_url` string pointing to GEE tile fetcher URL (`earthengine.googleapis.com`).

## 3. Pixel Hover Sampling Validation

Request:
`GET http://127.0.0.1:5000/api/sample?lat=13.42&lng=75.53&band=NDVI`

Expected Response:
```json
{
  "band": "NDVI",
  "value": 0.6234
}
```

## Related Documents
- [07_API_ARCHITECTURE.md](./architecture/07_API_ARCHITECTURE.md)
- [DOCUMENTATION_AUDIT_REPORT.md](./DOCUMENTATION_AUDIT_REPORT.md)

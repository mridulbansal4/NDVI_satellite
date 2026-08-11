# 23 — System Performance Evaluation

## Processing Benchmarks & Metrics

1. **Google Earth Engine Reduction Latency**:
   - Small Polygons (< 5 Hectares): ~2.5 - 4.0 seconds for complete median composite + 10m grid calculation.
   - Large Polygons (> 50 Hectares): Dynamic auto-coarsening scales grid cell size up from 10m to maintain target budget under `MAX_GRID_CELLS = 2000`, preventing HTTP timeout.

2. **Database Performance**:
   - PostgreSQL PostGIS GIST spatial indexing (`CREATE INDEX ON farms USING GIST(boundary_geom)`) maintains sub-millisecond polygon lookup times.
   - Connection pool management (`psycopg2.ThreadedConnectionPool(max=20)`) ensures multi-threaded request processing without connection exhaustion.

## Related Documents
- [11_DATABASE.md](../architecture/11_DATABASE.md)
- [24_LIMITATIONS.md](./24_LIMITATIONS.md)

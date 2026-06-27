# System Design — Satellite Agronomy Intelligence Platform

## 1. Why PostgreSQL + PostGIS (not MongoDB, not DynamoDB)?

Farm boundary data is inherently spatial — polygons in WGS84 coordinates — and PostGIS provides native GEOMETRY types, GIST indexes, and functions like `ST_GeomFromGeoJSON()` and `ST_Contains()` that make spatial queries first-class SQL. PostgreSQL's ACID compliance ensures that a multi-step onboarding transaction (farmer → farm → crop) never partially commits, which is critical when a mobile user loses connectivity between steps. DynamoDB lacks PostGIS support entirely, and MongoDB's geospatial capabilities are weaker than PostGIS for polygon-level operations like area intersection and spatial joins. PostgreSQL also supports strong enum types, TIMESTAMPTZ, and `DISTINCT ON` — all used heavily in this schema — with no application-layer workarounds.

## 2. Why Firestore only for sessions + alerts (not farm data)?

Firestore is used exclusively for ephemeral, real-time data that has a short TTL or needs push delivery to the frontend. Onboarding sessions (`/farmer_sessions`) are temporary — they exist only until Step 9 completes and are then deleted; storing them in PostgreSQL would require scheduled cleanup jobs. OTP tokens (`/otp_sessions`) need TTL-based expiry, which Firestore handles natively. Farm alerts (`/farm_alerts`) need to push dashboard updates to the React frontend reactively without the client polling a SQL endpoint every few seconds. Firestore's document listeners enable this. Storing actual farm, crop, and VI report data in Firestore would lose spatial query capability, schema enforcement, and relational join support — all of which the dashboard and VI engine depend on.

## 3. Exact data flow: mobile login → onboarding → PostgreSQL commit → VI Engine → dashboard

1. **Login**: `POST /auth/login` with mobile number → server checks `farmers` table; if new, inserts minimal row + writes `/farmer_sessions/{farmer_id}` to Firestore with `current_step: 1`.
2. **Steps 1–8**: Each step POSTs to the corresponding Flask blueprint → service validates → repository writes to PostgreSQL (`farmers`, `farmer_locations`, `farms`, `crops`, `irrigation`, `soil_info`). After each write, `write_session()` updates the Firestore session document with `current_step` and `partial_data` — enabling resume on reconnect.
3. **Step 9 (Consent)**: `POST /consent` → inserts `consents` row → calls `delete_session()` to remove Firestore document (onboarding committed) → logs "VI Engine triggered for farm_id: X" (mock in dev; real GEE job in production).
4. **VI Engine**: Reads `farms.boundary_geom` (PostGIS) → calls Google Earth Engine API → computes NDVI/EVI/SAVI/NDMI/NDWI/GNDVI → appends row to `vi_reports` (PostgreSQL) → immediately writes `/farm_alerts/{farm_id}` to Firestore.
5. **Dashboard**: `GET /dashboard` (JWT required) → joins `farmers + farms + crops + DISTINCT ON vi_reports` in PostgreSQL → returns assembled JSON. Frontend also subscribes to `/farm_alerts/{farm_id}` for real-time badge updates without re-fetching.

## 4. How does this scale to 100,000 farmers?

**Indexing**: All high-cardinality lookup columns are indexed (mobile, firebase_uid, farm_id, created_at DESC). The GIST index on `boundary_geom` makes spatial queries O(log n) instead of O(n). `DISTINCT ON (farm_id)` for latest VI reports is served from the composite index `(farm_id, created_at DESC)`.

**Connection pooling**: `psycopg2.ThreadedConnectionPool` with `max=20` connections handles Flask's multi-threaded request model. For 100K concurrent users, deploy behind Gunicorn (4–8 workers × 20 pool = 80–160 open connections) with PgBouncer in transaction mode to multiplex thousands of app connections onto ~50 PostgreSQL server connections.

**GEE batch jobs**: At 100K farmers, VI Engine runs as scheduled GEE batch tasks (one polygon per Earth Engine `Export.image.toCloudStorage` call) grouped by district to minimize API quota usage. A task queue (Cloud Tasks or Celery + Redis) dispatches weekly jobs per farm without blocking the API server.

**Horizontal scaling**: The Flask API is stateless (JWT auth, no server-side session storage). Deploy multiple replicas behind a load balancer (Cloud Run, Kubernetes). Firestore scales automatically. PostgreSQL can be vertically scaled or read-scaled with a read replica for dashboard queries.

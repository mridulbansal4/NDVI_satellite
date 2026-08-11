# 11 — Dual-Database Architecture

MindstriX utilizes two complementary database systems: **PostgreSQL + PostGIS** for persistent spatial and relational data, and **Cloud Firestore** for real-time ephemeral state.

---

## 1. Relational Database: PostgreSQL + PostGIS

- **Connection Pool**: `psycopg2.ThreadedConnectionPool` initialized in `backend/db/pool.py`.
- **Primary Schema Tables** (`schema.sql`):
  1. `organizations`: FPO, cooperative, or corporate accounts.
  2. `users`: System users linked to `firebase_uid`.
  3. `farmer_profiles`: Age, gender, onboarding step state.
  4. `farmer_locations`: PIN code, state, district, taluka, village.
  5. `farms`: Boundary stored as `GEOMETRY(POLYGON, 4326)`, area, ownership.
  6. `crops`: Crop type, variety, sowing date, season (`kharif`, `rabi`, `zaid`).
  7. `irrigation`: Irrigation method (rainfed, borewell, drip, etc.).
  8. `soil_info`: Soil type, pH, organic matter percentage.
  9. `consents`: Granted consents and timestamps.
  10. `vi_reports`: Vegetation analysis output reports and index statistics.
  11. `api_keys`: Hashed third-party API credentials.
  12. `api_request_logs`: Audit log for API usage.
  13. `webhooks`: Outbound event webhook subscriptions.

- **Spatial Indexing**:
  ```sql
  CREATE INDEX ON farms USING GIST(boundary_geom);
  CREATE INDEX ON vi_reports(farm_id, created_at DESC);
  ```

---

## 2. Real-Time Document Store: Cloud Firestore

- **Collections (`backend/firestore/`)**:
  - `/otp_sessions/{mobile}`: OTP tracking with TTL policy.
  - `/farmer_sessions/{user_id}`: Ephemeral onboarding step data (deleted upon Step 9 completion).
  - `/farm_alerts/{farm_id}`: Real-time analysis notification push targets for frontend listeners.
  - `/org_live_stats/{org_id}`: Organization-level live stats.

## Related Documents
- [08_BACKEND_ARCHITECTURE.md](./08_BACKEND_ARCHITECTURE.md)
- [17_CONFIGURATION.md](./17_CONFIGURATION.md)

# 16 — Error Handling & Resilience

## Error Handling Architectural Principles

1. **Explicit Failures**: The API does not return dummy 0-byte fallbacks or silent empty array mocks when an exception occurs.
2. **HTTP Status Code Mapping**:
   - `400 Bad Request`: Invalid GeoJSON geometry, invalid latitude/longitude arguments, missing required fields.
   - `401 Unauthorized`: Missing or invalid JWT ID token.
   - `404 Not Found`: Point sampling requested before running polygon analysis.
   - `500 Internal Server Error`: Uncaught backend execution exception.
   - `503 Service Unavailable`: Google Earth Engine not initialized or unauthenticated.

## Resilience & Degraded Modes

- **GEE Initialization Failures**: If `initialize_gee()` fails at server startup (e.g. missing `GEE_PROJECT_ID`), `app._gee_ready` is set to `False`. Non-GEE endpoints (health check, auth, DB CRUD) continue running normally.
- **PostgreSQL Pool Failure**: If PostgreSQL pool initialization fails (`init_pool()`), the server switches to GEE-only mode and logs a clear warning without crashing the Flask web server.

## Related Documents
- [07_API_ARCHITECTURE.md](./07_API_ARCHITECTURE.md)
- [08_BACKEND_ARCHITECTURE.md](./08_BACKEND_ARCHITECTURE.md)
- [25_KNOWN_ISSUES.md](../evaluation/25_KNOWN_ISSUES.md)

# Contributing Guidelines

Thank you for contributing to the MindstriX Satellite Agronomy Intelligence Platform.

## Core Architectural Rules

1. **GEE Isolation Requirement**:
   - All Google Earth Engine Python API calls **MUST** remain strictly within `backend/services/gee_service.py` and `backend/services/sar_service.py`.
   - Other service layers and Flask route handlers must only interact with returned `ee.Image` or `ee.Geometry` objects. Never invoke direct GEE API calls inside Flask routes or frontend code.

2. **Centralized Configuration**:
   - All threshold values, array limits, cloud cover tolerances, and index weights **MUST** be defined in `backend/config.py`.
   - Do not scatter magic numbers or arbitrary constants inside service logic.

3. **No Fallback Masking**:
   - Do not swallow errors with silent fallback empty data structures or fake values when GEE or database queries fail. Return appropriate HTTP error codes (e.g. 503 for GEE unavailable, 400 for invalid polygon).

4. **Zero Documentation Drift**:
   - If you modify an API route, update `docs/architecture/07_API_ARCHITECTURE.md`.
   - If you update a index computation formula, update `backend/services/index_service.py` AND `docs/architecture/06_VEGETATION_INDICES.md`.

## Related Documents
- [08_BACKEND_ARCHITECTURE.md](./architecture/08_BACKEND_ARCHITECTURE.md)
- [17_CONFIGURATION.md](./architecture/17_CONFIGURATION.md)

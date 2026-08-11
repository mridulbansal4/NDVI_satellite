# 24 — Technical Limitations

This document lists the technical constraints of the current implementation.

## Architectural Constraints

1. **In-Memory Chatbot Session Memory (`backend/chatbot/memory.py`)**:
   - Chatbot history is stored in an in-memory Python dictionary keyed by `session_id`.
   - In a multi-worker production deployment (e.g. Gunicorn with 4 workers), session memory is lost across different worker processes unless pinned or migrated to an external cache like Redis.

2. **Synchronous GEE Execution**:
   - GEE API calls execute synchronously during the Flask request handling thread (`composite.getInfo()`). Extremely large farm arrays could hit client HTTP timeouts.

3. **Cloud Cover Availability**:
   - Optical Sentinel-2 imagery relies on scenes matching `MAX_CLOUD_COVER_PCT <= 20%`. In monsoon seasons, optical imagery availability drops, necessitating fallback to Sentinel-1 SAR radar mode.

## Related Documents
- [23_PERFORMANCE.md](./23_PERFORMANCE.md)
- [25_KNOWN_ISSUES.md](./25_KNOWN_ISSUES.md)
- [26_FUTURE_WORK.md](../future/26_FUTURE_WORK.md)

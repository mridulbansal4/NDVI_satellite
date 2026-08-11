# 26 — Proposed Infrastructure Upgrades

This document outlines proposed infrastructure improvements based on analysis of the current backend architecture.

## Proposed Upgrades

1. **Redis Session Storage for Chatbot (`backend/chatbot/memory.py`)**:
   - Replace in-memory dictionary history with a Redis-backed session store to allow multi-worker horizontal scaling under Gunicorn or Kubernetes.

2. **Asynchronous Background Task Queue for GEE**:
   - Integrate Cloud Tasks or Celery + Redis to dispatch heavy Earth Engine batch jobs (`Export.image.toCloudStorage`) asynchronously for enterprise farm portfolios.

3. **PostgreSQL Read Replicas**:
   - Direct high-volume dashboard read queries (`GET /dashboard`) to dedicated read replicas to unburden the primary spatial write node.

## Related Documents
- [24_LIMITATIONS.md](../evaluation/24_LIMITATIONS.md)
- [27_ROADMAP.md](./27_ROADMAP.md)

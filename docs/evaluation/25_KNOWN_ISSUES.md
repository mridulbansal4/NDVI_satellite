# 25 — Known Technical Issues

## Known Edge Cases & Handling

1. **GEE Token Expiry During Long Server Runtime**:
   - If local GEE OAuth credentials in `~/.config/earthengine/credentials` expire while the server is active, subsequent requests return GEE Auth exception logs. Re-running `earthengine authenticate` resolves this.

2. **Ollama Connection Timeout**:
   - If the local Ollama LLM daemon is not running (`ollama serve`), `/chatbot/chat` returns an HTTP 502 error. The core satellite mapping dashboard remains fully functional.

3. **Sparse Scene Coverage in Small Lookback Windows**:
   - Single-day analysis (`/api/analyze-day`) can return zero scenes if Sentinel-2 had no orbital pass over the exact coordinates on that target date.

## Related Documents
- [16_ERROR_HANDLING.md](../architecture/16_ERROR_HANDLING.md)
- [24_LIMITATIONS.md](./24_LIMITATIONS.md)

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Satellite Agronomy Intelligence Platform ("MindstriX"). Users draw farm-field polygons on a Leaflet map; the backend pulls Sentinel-2 imagery from Google Earth Engine (GEE), computes vegetation indices, and returns a smoothed per-cell heatmap grid plus farm statistics. A chatbot ("Krishi Mitra") answers questions grounded in the current field's stats.

The backend is **Go**. It was ported from Python/Flask under `PRAGYA_GO_MIGRATION_PRD.md`; the Python implementation was deleted at tag `v2.0.0-go` and can be recovered with `git checkout v2.0.0-go~1 -- legacy-python`.

## Commands

```powershell
# Backend (Go, port 5000)
go build -o bin/server ./cmd/server
./bin/server                       # or: go run ./cmd/server
# Health check: GET http://127.0.0.1:5000/health → {"gee_ready": true, ...}

make build        # build the binary
make test         # go test
make test-race    # go test -race
make verify       # vet + staticcheck + race tests + contract suite  ← the gate
make contract     # replay the frozen HTTP corpus against a running server
make docker       # multi-stage distroless image

# Frontend (Vite dev server, port 5173) — unchanged by the migration
cd frontend
npm install       # first time only
npm run dev
npm run build     # production build → frontend/dist (Firebase hosting target)
npm run lint
```

`make verify` must be green before any change is considered done. It needs a server running on `:5001` for the contract stage (`make contract GO_BASE=...` to point elsewhere).

## Architecture

### `cmd/server` — entrypoint

Wiring only: config → dependencies → router → `ListenAndServe`. Startup probes for GEE and Firebase run **once, in goroutines** — neither may prevent startup, because `/health`, `/auth/*` and `/dashboard` must work without them. `WriteTimeout` is 300 s (a cold `/api/analyze` on a large polygon can exceed 60 s, and Vite proxies `/api` with a 300 s timeout); shutdown drains for 30 s.

### `internal/gee` — the only package that talks to Earth Engine

There is **no Go Earth Engine SDK**; Google ships JavaScript and Python clients only. The Python `ee` package is a lazy computation-graph builder, so this package reimplements that:

- `eeexpr/` builds Earth Engine expression-graph JSON by hand. `Compile` interns shared subtrees and hoists function-definition bodies (the wire format stores a body as a *string reference*). `Canonicalise` inlines a graph and alpha-renames mapping variables so two graphs can be compared for meaning rather than byte layout.
- `session.go` resolves credentials: a service-account key (`GEE_SERVICE_ACCOUNT_KEY`, the production path) or the `earthengine authenticate` refresh token plus `GEE_OAUTH_CLIENT_ID`/`SECRET` (developer fallback).
- `client.go` performs `value:compute`, `table:computeFeatures` (with `nextPageToken` paging) and `maps`. Retries **only** 429 and 5xx — a 400 means the expression graph is malformed and retrying just delays the real error.

**`X-Goog-User-Project` is required** with user credentials, or Earth Engine answers 403 "Not signed up for Earth Engine" even for a properly registered project.

### `internal/pipeline` — the analytics core

Never imports `net/http` or Gin; it deals in expression graphs and plain data, behind an `EEClient` interface. That layering is what makes the parity tests possible.

`sentinel2.go` → `indices.go` → `grid.go` → `smooth.go` → `stats.go`, plus the independent `sentinel1.go` / `radar_indices.go` / `radar_grid.go` radar path. `cache.go` is a TTL'd, bounded expression cache replacing the old process-global `app._last_indexed_image`.

`smooth.go` is a deliberately literal port: the centroid counts the duplicated closing vertex, `j == i` is included in the weighted sum, and iteration runs in slice order because float addition is not associative. Those quirks are load-bearing for numeric parity.

### `internal/httpapi` — the HTTP layer

Never builds expression graphs. All 24 routes, global CORS, three distinct error envelope shapes (`{"error":…}`, `{"errors":{field:[msg]}}`, `{"msg":…}`), and marshmallow-compatible validation.

### Other packages

`internal/config` (the single tuning surface), `internal/geo` (polygon validation), `internal/repo` + `internal/db` (hand-written SQL over pgx — no ORM, because the queries use PostGIS functions, `DISTINCT ON`, `ON CONFLICT … RETURNING` and `= ANY($1::uuid[])`), `internal/service` (onboarding, SMS, PIN lookup), `internal/crypto` (Werkzeug hash compatibility), `internal/jwtutil`, `internal/firebase`, `internal/firestore`, `internal/chatbot`, `internal/ollama`, `internal/logging`.

**Endpoints:** `/api/analyze`, `/api/analyze-dates`, `/api/analyze-day`, `/api/analyze-radar`, `/api/analyze-radar-dates`, `/api/sample`, `/api/auth/*`, `/auth/signup`, `/auth/login`, the 9-step onboarding (`/farmer/*`, `/farm`, `/crop`, `/irrigation`, `/soil`, `/consent`), `/dashboard`, `/chatbot/*`, `/health`. Full matrix with verified responses in [docs/ENDPOINT_VERIFICATION.md](docs/ENDPOINT_VERIFICATION.md).

**`internal/config/config.go` is the single tuning surface.** All GEE settings, band aliases, `CVIWeights` (must sum to 1.0), grid resolution, cloud thresholds, palettes, and the index→interpretation threshold tables live there. Change behaviour there, not in the service packages. Thresholds are **descending ordered slices**, not maps — the ordering is load-bearing.

Note: `config.go` weights/thresholds and the values quoted in `README.md` have drifted apart; **trust `config.go`** (see K2).

## Testing strategy

Four independent layers, all gated on fixtures captured from the original Python implementation before it was deleted:

| Layer | What | Where |
|---|---|---|
| 1 | EE expression graphs — every builder compared for structural equality | `internal/gee/eeexpr/testdata/`, asserted in `internal/pipeline/graph_test.go` |
| 2 | HTTP contract — 73 cases replayed against a running server | `testdata/golden/`, run by `tools/contract` |
| 3 | Numeric parity — full responses for five fixture polygons | `testdata/golden/numeric/` |
| 4 | Cross-runtime — Werkzeug hashes and prompt renders | `internal/crypto/testdata/`, `internal/chatbot/testdata/` |

Live Earth Engine tests are gated behind `GEE_LIVE_TEST=1` so the default suite is offline and deterministic.

**The fixtures beat any documentation, including the PRD.** Twelve of the PRD's stated Earth Engine function/argument names were wrong in ways that produce a runtime 400 from Google rather than a compile error. If you add an EE call, capture a fixture for it.

## Conventions

- Config over code: tune `internal/config/config.go` and `.env`.
- Keep Earth Engine calls confined to `internal/gee`; other packages pass `eeexpr.Image` / `eeexpr.Geometry` values around.
- **Never modify `frontend/`.** The HTTP contract is frozen: if a Go response breaks the UI, the Go response is wrong.
- Index values, means and samples are `*float64`. A masked cell must serialise as JSON `null`, never `0` — a `0` renders dark red on the heatmap and is a visible bug.
- `.env` and `serviceAccountKey.json` are gitignored — never commit secrets.
- Error strings are user-facing contract text, so they are capitalised and end with periods. `ST1005` is disabled repo-wide for this reason; see `staticcheck.conf`.

## Known issues

[docs/KNOWN_ISSUES.md](docs/KNOWN_ISSUES.md) lists 15 behaviours (K1–K15) that are **deliberately preserved** from the Python implementation, including two field-type asymmetries where the same database column serialises differently on different endpoints. Do not "fix" them without checking the frontend first. [docs/CHANGELOG.md](docs/CHANGELOG.md) records the migration.

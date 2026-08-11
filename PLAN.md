# PLAN — Project Pragya: Python → Go Backend Migration

Derived from `PRAGYA_GO_MIGRATION_PRD.md`. Every task cites the PRD section it
comes from. Phases are strictly ordered: no phase starts while the previous one
is red (PRD §14).

Legend: `[x]` done and verified · `[~]` partially done, see notes · `[ ]` not started

---

## Phase 0 — Groundwork  ✅

**Acceptance:** `go build ./...` succeeds; goldens committed; the Python backend
still runs from its new path.

- [x] `go mod init github.com/SanTiwari07/NDVI_satellite` (Go 1.23) — §3.2
- [x] `Makefile` with a `verify` gate, `.golangci.yml`, `staticcheck.conf`, multi-stage `Dockerfile` — §12.5, §13.10
- [x] `git mv backend legacy-python`; delete the dead `legacy/` CLI — §1.3, §14
- [x] `internal/config/config.go` — 1:1 port of `config.py` + `chatbot/config.py` + `app.py` palettes — §9.1
- [x] `internal/config/config_test.go` — independent second transcription of Appendix A
- [x] `.gitignore` += `gee-service-account.json`, `bin/`; `.env.example` += Go keys — §9.2
- [x] Capture Layer-1 EE graph fixtures (37) — §5.4
- [x] Capture the live `maps` request/response — §5.2, §5.6
- [x] Capture Layer-2 contract goldens (73 cases) — §12.2
- [x] Capture Layer-3 numeric goldens (5 polygons × S2 + S1) — §12.3
- [x] Capture werkzeug hash fixtures (9) — §8.2
- [x] Capture chatbot prompt renders (2) — §10.11
- [x] File K1–K10 + K11/K12 in `docs/KNOWN_ISSUES.md` — §13.9

## Phase 1 — HTTP skeleton, no GEE  ✅

**Acceptance:** contract runner passes E1 and every 400/401/422/503 branch.

- [x] `internal/geo` — `ValidatePolygon`, exact wording/ordering/number formatting — §6.1
- [x] `internal/logging` — slog handler matching the Python log format — §9.4
- [x] `internal/httpapi/errors.go` — the three envelope shapes + marshmallow translation — §7.11
- [x] `internal/httpapi/schemas.go` — every marshmallow schema — §6.2
- [x] `internal/httpapi/middleware/cors.go` — global CORS — §9.3
- [x] `internal/httpapi/middleware/jwt.go` — the four failure branches — §8.3
- [x] `internal/httpapi/router.go` — all 24 routes — §10.1
- [x] Analysis + onboarding + chatbot handlers, validation live, service tail 501
- [x] `cmd/server/main.go` — wiring, one-shot probes, timeouts, graceful drain — §3.1, §13.3
- [x] `tools/contract` — the Layer-2 runner — §12.2

## Phase 2 — Earth Engine core  (critical path)

**Acceptance:** `go test ./internal/gee/...` green offline; a live smoke test
computes `s2_scene_count` and matches the Python number exactly.

- [x] `eeexpr` node types + `Compile` with interning and funcdef-body hoisting — §5.3, §11.1
- [x] Canonicaliser + `assertGraphEqualsGolden` (semantic, not textual) — §5.4
- [x] `MapOverCollection` with `_MAPPING_VAR_<depth>_<idx>` naming — §5.8
- [x] Typed wrappers: Image, ImageCollection, Geometry, Projection, Reducer, Filter — §11.1
- [x] All graph builders for G1–G24, each gated by its fixture — §5.7
- [x] `internal/gee/session.go` — service-account primary, refresh-token fallback — §5.5
- [x] `internal/gee/client.go` — computeValue / computeFeatures / createMap — §11.2
- [x] `internal/gee/errors.go` — classification + retry policy (429/5xx only) — §13.4
- [x] `computeFeatures` pagination via `nextPageToken` — §13.6
- [x] Live smoke test against the fixture polygon

## Phase 3 — Sentinel-2 pipeline

**Acceptance:** Layer-3 numeric parity green on the fixture polygons; E2/E3/E4/E7 wired.

- [x] `pipeline/sentinel2.go` — composite, dates, single-day — §7.1, §7.2
- [x] `pipeline/indices.go` — six indices + CVI, EVI/SAVI arithmetic rewrite — §7.3
- [x] `pipeline/grid.go` — `coveringGrid` + auto-coarsening loop — §7.4
- [x] `pipeline/smooth.go` — Gaussian smoothing, exact port — §7.6
- [x] `pipeline/stats.go` — means, stdDev, histogram, confidence — §7.8
- [x] `pipeline/interpret.go` + `palettes.go` — §7.5, §5.6
- [x] `pipeline/cache.go` — TTL'd `ExprCache` replacing `app._last_indexed_image` — §7.7
- [x] Wire E2, E3, E4, E7
- [x] Numeric parity test harness — §12.3

## Phase 4 — Sentinel-1 radar pipeline

**Acceptance:** radar numeric parity green; E5/E6 wired.

- [x] `pipeline/sentinel1.go` — base collection, speckle filter, composite windows — §7.9
- [x] `pipeline/radar_indices.go` — SMI, RVI, RATIO — §7.9
- [x] `pipeline/radar_grid.go` — 5-band reduction, smooth all five, moisture class — §7.9
- [x] Wire E5, E6

## Phase 5 — Data layer, auth, onboarding

**Acceptance:** a Python-created user with a Python-issued JWT logs in and loads
`/dashboard` against the Go server.

- [x] `internal/db/pool.go` — pgxpool, min 2 / max 20 — §13.5
- [x] `internal/crypto/werkzeug.go` — scrypt + pbkdf2 verify/generate — §8.2
- [x] `internal/jwtutil` — Flask-JWT-Extended-compatible issue path — §8.3
- [x] `internal/repo/*` — seven repositories, hand-written SQL — §2.3
- [x] `internal/service/*` — onboarding business logic + Firestore mirroring — §7.10
- [x] `internal/service/pincode.go` — India Post client (K11 fix accepted)
- [x] `internal/service/sms.go` — OTP store, janitor, gateway — §8.5
- [x] `internal/firebase/admin.go` + `internal/firestore/*` — §8.4, §13.7
- [x] Wire E8–E21

## Phase 6 — Chatbot

**Acceptance:** prompt-render diff test green.

- [x] `internal/chatbot/prompt.go` — `text/template`, character-for-character — §10.11
- [x] `internal/chatbot/memory.go` — mutex-guarded, capped, trim-from-front — §10.11
- [x] `internal/ollama/client.go` — direct `/api/chat`, LangChain dropped — §10.11
- [x] Wire E22–E24

## Phase 7 — Cutover & cleanup

- [x] Full contract suite green
- [x] Endpoint-by-endpoint verification, actual status + body recorded
- [x] Docs updated (`CLAUDE.md`, `README.md`, `docs/`) — §16
- [x] `REPORT.md`
- [ ] Delete `legacy-python/` — **deliberately NOT done**, see REPORT.md

---

## Endpoint checklist (§10.1)

Verification columns: H = happy path · I = missing/invalid input · A = auth/unauthorized · E = edge cases.

| # | Method | Path | Auth | H | I | A | E |
|---|---|---|---|---|---|---|---|
| E1 | GET | `/health` | none | [x] | n/a | n/a | [x] |
| E2 | POST | `/api/analyze` | none | [x] | [x] | n/a | [x] |
| E3 | POST | `/api/analyze-dates` | none | [x] | [x] | n/a | [x] |
| E4 | POST | `/api/analyze-day` | none | [x] | [x] | n/a | [x] |
| E5 | POST | `/api/analyze-radar-dates` | none | [x] | [x] | n/a | [x] |
| E6 | POST | `/api/analyze-radar` | none | [x] | [x] | n/a | [x] |
| E7 | GET | `/api/sample` | none | [x] | [x] | n/a | [x] |
| E8 | POST | `/api/auth/verify-token` | none | [x] | [x] | [x] | [x] |
| E9 | POST | `/api/auth/send-otp` | none | [~] | [x] | n/a | [~] |
| E10 | POST | `/api/auth/verify-otp` | none | [~] | [x] | n/a | [x] |
| E11 | POST | `/auth/signup` | none | [x] | [x] | n/a | [x] |
| E12 | POST | `/auth/login` | none | [x] | [x] | n/a | [x] |
| E13 | POST | `/farmer/basic-details` | JWT | [x] | [x] | [x] | [x] |
| E14 | POST | `/farmer/location` | JWT | [x] | [x] | [x] | [x] |
| E15 | GET | `/farmer/pincode/:pin` | none | [x] | [x] | n/a | [x] |
| E16 | POST | `/farm` | JWT | [x] | [x] | [x] | [x] |
| E17 | POST | `/crop` | JWT | [x] | [x] | [x] | [x] |
| E18 | POST | `/irrigation` | JWT | [x] | [x] | [x] | [x] |
| E19 | POST | `/soil` | JWT | [x] | [x] | [x] | [x] |
| E20 | POST | `/consent` | JWT | [x] | [x] | [x] | [x] |
| E21 | GET | `/dashboard` | JWT | [x] | n/a | [x] | [x] |
| E22 | POST | `/chatbot/chat` | none | [~] | [x] | n/a | [x] |
| E23 | POST | `/chatbot/reset` | none | [x] | [x] | n/a | [x] |
| E24 | GET | `/chatbot/health` | none | [x] | n/a | n/a | [x] |

`[~]` = cannot be exercised in this environment; reason recorded in REPORT.md
(E9 would send a real billed SMS; E22 needs a running Ollama server).

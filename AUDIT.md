# Codebase Audit — MindstriX / PRAGYA Satellite Agronomy Platform

Audit date: 2026-08-12. Branch `main_go`, HEAD `e304c93`.

The audit itself was read-only: every finding cites code that was opened and
read, and claims that could not be traced to a specific line were dropped. The
fixes recorded in **Remediation status** below were applied afterwards, as a
separate step, on top of the audited commit.

## Remediation status

Fixed (`vet`, `staticcheck`, `go test -race` clean; contract suite unchanged at
68 pass / 0 fail / 5 skipped):

| ID | What changed |
|---|---|
| C1 | `Validate()` refuses to start when `FLASK_ENV=production` and the key is empty, still the built-in constant, or under 32 chars; a warning fires outside production. `DevJWTSecret` is now a named constant that says what it is. |
| H1 | Wrong OTP guesses count against `OTP_MAX_ATTEMPTS` (default 5) and discard the code; comparison is constant-time. The response is unchanged, so the contract still holds. |
| H2 | `firebase` and `firestore` construct their clients with `context.Background()` via a local `clientContext()`, mirroring the fix and commentary already in `gee/session.go`. |
| H3 | `middleware.BodyLimit` wraps every request body in `http.MaxBytesReader` (`MAX_REQUEST_BYTES`, default 8 MiB). |
| H4/H5 | `middleware.RateLimiter` (per-IP token bucket) on `/api`, `/auth` and `/chatbot`; `middleware.InFlight` bounds concurrent Earth Engine analyses. `/health` is exempt. |
| H6 | `Memory` gained a TTL and an LRU session bound, matching `ExprCache`. |
| M1 | `CORS_ORIGINS` env override, localhost list as default. |
| M3 | `schema.sql` weight and confidence-scale comments corrected to match `config.go`. |
| M5 | `.dockerignore` added — secrets, `frontend/`, build output and VCS metadata excluded from the build context. |
| M6 | Dockerfile base image bumped to `golang:1.25-alpine`. |
| M8 | Phone numbers are masked in logs (`91XXXXXX3210`). |
| M11 | `defer smsSvc.Close()` stops the OTP janitor at shutdown. |
| M12 | Request logs carry a correlation id (honouring an inbound `X-Request-Id`, sanitised) and a duration; the id is echoed in the response header. |
| L1 | Salt generation uses rejection sampling instead of a biased `% 62`. |
| L2 | Constant-time OTP comparison (folded into H1). |
| L5 | Dead code removed: `AnalyzeRequest`, the `rawJSON` alias, and the `lastBody` wrapper. `Memory.Sessions()` was kept — it is now used by the eviction tests. |
| M9 (partial) | New tests: rate limiter, in-flight gate, body cap, memory TTL/LRU, OTP attempt budget, `maskPhone`, `E164`, and the production secret rules. `middleware` and `service` moved off 0%. |

Not fixed, and why:

| ID | Reason |
|---|---|
| H7 | `/api/sample`'s global `__last__` slot is documented as intentional-for-parity (K6) and the frontend sends no field identifier, so keying it per user needs either a frontend change (frozen) or a session cookie. Product decision. |
| M2 | Needs a DB migration (`UNIQUE(farmer_id)` on `farmer_locations`) that must be applied to live data before the `ON CONFLICT` code can ship. Applying it blind was out of scope for this pass. |
| M4 | Adopting a migration runner is a project, not a patch. |
| M7 | The SMS gateway's documented API takes credentials as query parameters; changing that requires confirming what else the vendor supports. |
| M10 | The N+1 is deliberate parity (K7), slated for the Phase 8 batch. |
| L3 | `round`'s rounding mode is dormant until something writes `vi_reports`, which nothing does. |
| L4 | Fixed as part of this pass — docs now say Go 1.25. |

---

## System Overview

A satellite agronomy platform. A React/Leaflet frontend lets a user draw a
farm-field polygon; a Go backend pulls Sentinel-2 (optical) and Sentinel-1 (SAR)
imagery from Google Earth Engine, computes vegetation indices per grid cell,
smooths them, and returns a GeoJSON heatmap plus farm-level statistics. A
chatbot ("Krishi Mitra") answers questions grounded in the current field's stats.
A separate nine-step onboarding flow persists farmers, farms, crops, irrigation,
soil and consent to PostgreSQL/PostGIS.

The backend was ported from Python/Flask to Go under
`PRAGYA_GO_MIGRATION_PRD.md`; the Python implementation was deleted at tag
`v2.0.0-go`. The HTTP contract is deliberately frozen — the Go responses are
byte-compared against fixtures captured from the running Flask app.

### Entry points and build

| | |
|---|---|
| Backend entry | [cmd/server/main.go](cmd/server/main.go) — config → deps → router → `ListenAndServe`, port 5000 |
| Build | `go build -o bin/server ./cmd/server`, or `make build` |
| Gate | `make verify` = `vet` + `staticcheck` + `go test -race` + contract replay ([Makefile:96](Makefile:96)) |
| Frontend | Vite dev server on 5173, proxies `/api` only ([frontend/vite.config.js](frontend/vite.config.js)) |
| Container | [Dockerfile](Dockerfile) — multi-stage, distroless final image |

### Stack

Go 1.25 (`go.mod:3`; note the drift flagged in M6/L4), Gin 1.12, pgx v5.10,
golang-jwt v5.3, validator/v10.30, godotenv, `golang.org/x/crypto` (scrypt +
pbkdf2), firebase-admin-go v4.21, `cloud.google.com/go/firestore` v1.25,
`golang.org/x/oauth2`. No ORM — hand-written SQL, because the queries use
PostGIS functions, `DISTINCT ON`, `ON CONFLICT … RETURNING` and
`= ANY($1::uuid[])`.

There is **no Go Earth Engine SDK**. `internal/gee/eeexpr` hand-builds Earth
Engine expression-graph JSON (interning shared subtrees, hoisting function
bodies into string references) and posts it to `value:compute`,
`table:computeFeatures` and `maps`.

### Layering

```
cmd/server ──> internal/httpapi ──> internal/pipeline ──> internal/gee
                    │                  (EEClient iface)
                    ├──> internal/service ──> internal/repo ──> internal/db
                    ├──> internal/chatbot ──> internal/gemini | internal/ollama
                    └──> internal/firebase, internal/firestore
```

The layering is real and enforced by convention: `internal/pipeline` never
imports `net/http` or Gin and talks to Earth Engine through an `EEClient`
interface ([internal/pipeline/analyze.go:19-23](internal/pipeline/analyze.go:19)),
which is what makes the offline graph-parity tests possible. `internal/httpapi`
never builds expression graphs.

### Config and secrets

`internal/config/config.go` is the single tuning surface — `os.Getenv` is read
nowhere else. `.env` is loaded via godotenv from the working directory and the
repo root ([config.go:335-344](internal/config/config.go:335)). Secrets come
from environment variables (`DATABASE_URL`, `JWT_SECRET_KEY`, `GEMINI_API_KEY`,
`SMS_*`) plus two on-disk credential files (`serviceAccountKey.json`,
`gee-service-account.json`). Both files and `.env` are gitignored, and
`git ls-files` confirms none is tracked.

### Tests

Four layers: EE expression graphs (`internal/gee/eeexpr/testdata/`), HTTP
contract (74 goldens in `testdata/golden/`, replayed by `tools/contract`),
numeric parity (`testdata/golden/numeric/`), and cross-runtime fixtures
(Werkzeug hashes, prompt renders). Live Earth Engine tests are gated behind
`GEE_LIVE_TEST=1`. Run with `make test` / `make test-race`; the contract stage
needs a server running (`make contract GO_BASE=…`).

### What is genuinely unclear

- **`vi_reports` is never written.** The read path exists
  ([repo/onboarding.go:189](internal/repo/onboarding.go:189)) and `/dashboard`
  depends on it, but nothing in the Go tree inserts a row, and the code comment
  says so. Whether a separate writer is planned or was lost in the migration is
  not determinable from this repo.
- **Deployment topology.** `firebase.json` and `.firebaserc` exist and the
  frontend builds to a Firebase Hosting target, but there is no manifest for the
  Go backend (no Cloud Run/App Engine config, no compose file), so where the
  backend runs in production is not stated anywhere in the repo. Several
  findings below (M1, M5, C1) depend on that answer.

---

## Findings

| ID | Sev | Area | Location | Issue | Recommended fix | Effort |
|---|---|---|---|---|---|---|
| C1 | Critical | Security | [config.go:285](internal/config/config.go:285) | JWT signing key falls back to the hardcoded literal `dev-secret-change-me`; nothing validates it | Fail startup when `JWT_SECRET_KEY` is unset/short in non-dev env | S |
| H1 | High | Security | [sms.go:166-185](internal/service/sms.go:166) | OTP verification has no attempt counter and no rate limit; a wrong guess leaves the record intact | Delete after N failures; add per-phone and per-IP rate limits | S |
| H2 | High | Correctness | [firebase/admin.go:50-79](internal/firebase/admin.go:50), [firestore/client.go:56-86](internal/firestore/client.go:56) | `sync.Once` captures the first caller's context and reuses it for the process lifetime — the exact bug already fixed in `gee/session.go` | Use `context.Background()` for client construction, as `clientContext()` does | S |
| H3 | High | Security | [analyze_handler.go:62](internal/httpapi/analyze_handler.go:62) | Request bodies are read with `io.ReadAll` and no size cap, on unauthenticated routes | Wrap with `http.MaxBytesReader`; cap polygon vertex count | S |
| H4 | High | Security | [router.go:119-130](internal/httpapi/router.go:119), [router.go:167-172](internal/httpapi/router.go:167) | Every analysis and chatbot route is unauthenticated and unthrottled; each call spends Earth Engine quota or Gemini tokens | Require JWT, or add per-IP rate limiting and a global concurrency cap | M |
| H5 | High | Security | [werkzeug.go:69](internal/crypto/werkzeug.go:69), [onboarding_handler.go:114](internal/httpapi/onboarding_handler.go:114) | Unauthenticated `/auth/signup` runs scrypt N=32768 (~32 MB) per request and creates a row, with no throttle | Rate-limit auth routes; bound concurrent password hashing | S |
| H6 | High | Performance | [memory.go:23,56](internal/chatbot/memory.go:23) | Chat sessions map is never evicted; `session_id` is caller-supplied | Add TTL/LRU eviction, matching `ExprCache` | S |
| H7 | High | Security | [cache.go:22](internal/pipeline/cache.go:22), [analyze_handler.go:254](internal/httpapi/analyze_handler.go:254) | `/api/sample` serves the most recent analysis to any unauthenticated caller — cross-tenant read | Key the cache per session/user instead of the `__last__` sentinel | M |
| M1 | Medium | Operability | [config.go:159-165](internal/config/config.go:159) | CORS origins hardcoded to five localhost ports with no env override | Read from `CORS_ORIGINS` with the localhost list as default | S |
| M2 | Medium | Data integrity | [farmer.go:119](internal/repo/farmer.go:119), [schema.sql:57](schema.sql:57) | `farmer_locations` has no unique constraint and the insert is unconditional; re-running step 3 duplicates rows | Add `UNIQUE(farmer_id)` + `ON CONFLICT DO UPDATE`, as `soil_info` does | S |
| M3 | Medium | Data integrity | [schema.sql:149-156](schema.sql:149) | Schema comments state 0.35/0.25/0.15/0.15/0.10 weights and "0–100 scale" confidence; the code uses 0.70/0.10/0.05/0.10/0.05 and 0–1 | Correct the comments to match `config.go` | S |
| M4 | Medium | Data integrity | [schema.sql](schema.sql), [migrate_add_password.sql](migrate_add_password.sql) | No migration tooling — one full schema dump plus one ad-hoc ALTER, unversioned and unordered | Adopt a migration runner (goose/atlas) and check in ordered migrations | M |
| M5 | Medium | Security | [Dockerfile:12](Dockerfile:12) | No `.dockerignore`; `COPY . .` pulls `.env`, `serviceAccountKey.json` and `frontend/node_modules` into build layers | Add `.dockerignore` excluding secrets, `frontend/`, `bin/`, `*.log` | S |
| M6 | Medium | Dependencies | [Dockerfile:9](Dockerfile:9) vs [go.mod:3](go.mod:3) | Image pins `golang:1.23-alpine` but the module requires `go 1.25.0` | Bump the base image to 1.25 | S |
| M7 | Medium | Security | [sms.go:127-137](internal/service/sms.go:127) | SMS gateway username and password are sent as URL query parameters over GET | Move to POST body or header auth if the gateway supports it; otherwise document the exposure | S |
| M8 | Medium | Security | [sms.go:158](internal/service/sms.go:158), [onboarding.go:176](internal/service/onboarding.go:176), [logging.go:46](internal/logging/logging.go:46) | Phone numbers, farmer ids and PIN codes are logged in cleartext to an unrotated file | Mask identifiers; add rotation or log only to stdout and let the platform handle it | S |
| M9 | Medium | Testing | coverage run (below) | 0% coverage on `service`, `repo`, `gemini`, `ollama`, `middleware`, `jwtutil`, `db`, `firebase`, `firestore` | Unit-test OTP, ownership checks, JWT middleware and the Gemini client | L |
| M10 | Medium | Performance | [dashboard.go:79](internal/service/dashboard.go:79) | One crops query per farm (N+1), knowingly preserved as K7 | Batch with `WHERE farm_id = ANY($1::uuid[])`, as the vi_reports query already does | S |
| M11 | Medium | Operability | [main.go:110](cmd/server/main.go:110) | `SMSService.Close()` is never called; the janitor goroutine outlives shutdown | Call `defer smsSvc.Close()` | S |
| M12 | Medium | Operability | [router.go:179-188](internal/httpapi/router.go:179) | Request log has no request id, latency, or client identity — only method/path/status | Add a request id and duration; correlate with the pipeline stage logs | S |
| L1 | Low | Security | [werkzeug.go:167](internal/crypto/werkzeug.go:167) | `saltAlphabet[int(b)%62]` is modulo-biased (256 mod 62 ≠ 0) | Use `rand.Int` over the alphabet length, or rejection sampling | S |
| L2 | Low | Security | [sms.go:180](internal/service/sms.go:180) | OTP compared with `!=` rather than a constant-time compare | Use `subtle.ConstantTimeCompare` | S |
| L3 | Low | Correctness | [repo.go:124-127](internal/repo/repo.go:124) | `round` is half-away-from-zero; Python's `round` is banker's rounding | Match Python's semantics, or document that it is dormant until `vi_reports` is written | S |
| L4 | Low | Dev experience | [README.md:462](README.md:462), [PLAN.md:16](PLAN.md:16) | Docs say "Go 1.23+"; the module requires 1.25.0 | Update the docs | S |
| L5 | Low | Dev experience | [schemas.go:122-125](internal/httpapi/schemas.go:122), [memory.go:67](internal/chatbot/memory.go:67), [analyze_handler.go:77-79](internal/httpapi/analyze_handler.go:77) | Dead code: `AnalyzeRequest` and `Memory.Sessions()` have no callers; `lastBody` is a one-line alias for `readBody` | Delete | S |

Measured coverage (`go test ./cmd/... ./internal/... ./tools/... -cover`):

```
crypto 83.3%   geo 79.0%   config 52.8%   httpapi 49.2%   eeexpr 48.9%
chatbot 43.9%  pipeline 13.9%
service, repo, gemini, ollama, middleware, jwtutil, db, firebase,
firestore, logging, gee, cmd/server, tools/*  ........... 0.0%
```

---

## Critical and High — detail

### C1 — JWT signing key silently defaults to a public constant

**What the code does.** `config.go:285` reads the signing key as
`JWTSecret: envStr("JWT_SECRET_KEY", "dev-secret-change-me")`. `envStr` returns
the default whenever the variable is unset *or* whitespace-only
([config.go:404-409](internal/config/config.go:404)). `Config.Validate()`
([config.go:366-387](internal/config/config.go:366)) checks CVI weights and
threshold ordering — it does not look at the secret. The value flows into
`middleware.JWT([]byte(d.Cfg.JWTSecret))` at
[router.go:113](internal/httpapi/router.go:113) and into the token issuer at
[main.go:127](cmd/server/main.go:127).

**Why it is wrong.** The fallback is not a placeholder that fails loudly — it is
a fully working key. A deployment that forgets `JWT_SECRET_KEY` starts
successfully, logs nothing unusual, serves `/health` with `status: ok`, and
issues and accepts tokens signed with a string that is committed to a public
repository.

**What an attacker does.** The verification path is deliberately liberal: any
HS256 token that parses, has not expired and carries a non-empty string `sub` is
accepted, and `sub` is used directly as the farmer id
([middleware/jwt.go:60-86](internal/httpapi/middleware/jwt.go:60)). No `type`,
`fresh` or `jti` claim is required. So an attacker signs
`{"sub":"<any farmer uuid>","exp":<future>}` with `dev-secret-change-me` and
reads or writes that farmer's data through `/dashboard`, `/farm`, `/crop`,
`/irrigation`, `/soil` and `/consent`. Farm ids are UUIDs, but the farmer id is
returned in the signup/login response body
([service/onboarding.go:52-56](internal/service/onboarding.go:52)), and
`/dashboard` enumerates every farm for whatever `sub` is presented — so one
self-registered account is enough to learn the response shape, and any leaked or
guessed farmer uuid is then fully impersonable. This is total authentication
bypass for every account on the deployment.

**Fix.** In `Validate()`, reject the default: if `Env` is not a development
value, return an error when `JWTSecret` is empty, equal to the fallback, or
shorter than 32 bytes. Startup failure is the correct behaviour here — unlike
GEE and Firebase, whose absence is a documented degraded mode, a missing signing
key is not a degraded mode, it is an open door.

### H1 — OTP can be brute-forced

**What the code does.** `VerifyOTP` ([sms.go:166-185](internal/service/sms.go:166))
looks up the record for the phone number, deletes it if expired, returns `false`
if the code does not match — **without deleting the record or counting the
attempt** — and deletes it only on success. The code is six digits
([sms.go:100-106](internal/service/sms.go:100)) with a 600-second TTL
([config.go:305](internal/config/config.go:305)). `POST /api/auth/verify-otp` is
unauthenticated ([router.go:129](internal/httpapi/router.go:129)) and there is no
rate-limiting middleware anywhere in the router
([router.go:107-113](internal/httpapi/router.go:107)).

**Why it is wrong.** A single-use code is only single-use on the *success* path.
Failures are free and unlimited, which turns a 10-minute window into an
unbounded guessing budget against a 900,000-value space.

**What an attacker does.** Requests an OTP for a target phone number, then
sprays `/api/auth/verify-otp` for that number. At even a modest 1,000 requests/s
the whole space is exhausted in about 15 minutes; the expected hit is ~7.5
minutes, inside the TTL. Success returns `{"ok":true,"phone":"+91…"}`, which is
the frontend's proof that the caller controls that number — so the attacker
completes onboarding against a phone number they do not own. The OTP path issues
no JWT, so this is phone-verification bypass rather than direct account
takeover; combined with C1 it is worse.

**Fix.** Store an attempt counter alongside the code; delete the record after
5 failures. Add per-phone and per-IP rate limits on both `send-otp` (which costs
money per message, see H4) and `verify-otp`.

### H2 — Firebase and Firestore cache a dead context, same bug already fixed in `gee`

**What the code does.** `Admin.client` ([firebase/admin.go:50-79](internal/firebase/admin.go:50))
builds the Firebase app inside `sync.Once` using **the first caller's** `ctx`:
`firebase.NewApp(ctx, conf, …)` then `app.Auth(ctx)`. `firestore.Client.DB`
([firestore/client.go:56-86](internal/firestore/client.go:56)) does the same.
The first caller of `Admin` is the startup probe at
[main.go:137-144](cmd/server/main.go:137), which creates a 30-second context and
`defer cancel()`s it as soon as the goroutine returns. For Firestore the first
caller is a request handler, whose context is cancelled when the response is
written.

**Why it is wrong.** These constructors capture the context for the lifetime of
the client, not just for construction — the credential/token source reuses it for
every later refresh. `internal/gee/session.go` documents this precisely and
fixes it: `clientContext()` returns `context.Background()` and is used at all
three capture points, with the comment "THREE places capture a context and reuse
it for every future token refresh"
([gee/session.go:242-264](internal/gee/session.go:242), used at lines 218, 236,
287, 333). That bug was found in production — Earth Engine access died about an
hour after startup. The identical pattern is still live in `firebase` and
`firestore`.

**What an unlucky input does.** The cached `auth.Client` holds an already-cancelled
context. Once the initial access token expires (~1 hour), the refresh runs
against a dead context and fails, so `POST /api/auth/verify-token` starts
returning 401 `Invalid or expired authentication token`
([onboarding_handler.go:55-66](internal/httpapi/onboarding_handler.go:55)) for
every user, indefinitely, until the process is restarted. Because `sync.Once`
also caches the *error*, a failure during that 30-second startup window disables
Firebase for the whole process lifetime. The failure is invisible in testing:
every test runs inside the first hour, which is exactly why the `gee` instance of
this bug survived the entire migration.

**Fix.** Construct both clients with `context.Background()` (per-call contexts
still apply to `VerifyIDToken` and the Firestore writes). Consider exporting
`clientContext()` or documenting the rule once, since this is now the third
occurrence.

### H3 — Unbounded request-body read on unauthenticated routes

**What the code does.** `readBody` does `io.ReadAll(c.Request.Body)` with no
limit ([analyze_handler.go:62](internal/httpapi/analyze_handler.go:62)), caches
the parsed map in the Gin context, and is the entry point for `/api/analyze`,
`/api/analyze-day`, `/api/analyze-radar`, `/chatbot/chat` and the OTP routes.
The server sets `ReadTimeout: 30s` ([main.go:153](cmd/server/main.go:153)) but no
`MaxBytesReader` anywhere. `geo.ValidatePolygon` then iterates the outer ring
with no cap on vertex count
([geo/geo.go:82-108](internal/geo/geo.go:82)) — it checks the ring has *at
least* 4 positions, never at most.

**Why it is wrong.** `ReadTimeout` bounds time, not bytes. A client on a fast
link can push hundreds of megabytes within 30 seconds, and the whole body is
held in memory twice (raw slice plus decoded map).

**What an attacker does.** A handful of concurrent POSTs with large bodies to
`/api/analyze` — no credentials needed — drives the process to OOM. A subtler
variant sends a syntactically valid polygon with millions of vertices: it passes
validation, is marshalled into an Earth Engine expression graph
([pipeline/analyze.go:491-506](internal/pipeline/analyze.go:491)) and shipped to
Google, burning quota per request.

**Fix.** `c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)`
before reading, and add an explicit upper bound on ring length in
`ValidatePolygon`. Note the second change alters a contract-frozen validation
path, so it needs a golden update.

### H4 — Every expensive endpoint is unauthenticated and unthrottled

**What the code does.** The `/api` group registers six analysis routes and
`/api/sample` with no middleware beyond CORS and logging
([router.go:119-130](internal/httpapi/router.go:119)); `jwtAuth` is applied only
to `/farmer/*`, the five onboarding POSTs and `/dashboard`
([router.go:142-164](internal/httpapi/router.go:142)). The `/chatbot` group is
likewise open ([router.go:167-172](internal/httpapi/router.go:167)).

**Why it is wrong.** These are the only routes in the system that cost money per
call. A single `/api/analyze` performs a scene count, an auto-coarsening grid
search that round-trips per candidate scale
([analyze.go:259-276](internal/pipeline/analyze.go:259)), a `computeFeatures`
over up to 2,000 cells, three statistics calls, and **seven** `CreateMap` calls
([analyze.go:121-135](internal/pipeline/analyze.go:121)) — well over a dozen
Earth Engine requests, taking ~26 s wall-clock in observed runs. Each
`/chatbot/chat` bills Gemini tokens
([gemini/client.go:130](internal/gemini/client.go:130)).

**What an attacker does.** Scripts `/api/analyze` in a loop and exhausts the
project's Earth Engine quota, which takes the platform down for every real user,
or runs up the Gemini bill via `/chatbot/chat`. `/api/auth/send-otp` is the same
class: every call sends a real SMS with a real per-message cost
([sms.go:111-160](internal/service/sms.go:111)).

**Fix.** Shortest path is per-IP rate limiting plus a global concurrency
semaphore on the analysis routes. Requiring a JWT would be stronger but changes
the frontend's call pattern, which is frozen — check `frontend/src/api.js`
before choosing.

### H5 — Unauthenticated signup performs a 32 MB scrypt and creates a row

**What the code does.** `POST /auth/signup` validates, then calls
`crypto.GeneratePassword`, which derives with `scrypt:32768:8:1` and dkLen 64
([werkzeug.go:64-75](internal/crypto/werkzeug.go:64), [werkzeug.go:97](internal/crypto/werkzeug.go:97)).
scrypt with N=32768, r=8 allocates 128·N·r ≈ 32 MB per invocation. The route has
no auth and no throttle ([router.go:135](internal/httpapi/router.go:135)).
`/auth/login` reaches the same cost via `VerifyPassword` whenever the mobile
number exists.

**Why it is wrong.** The memory-hardness that makes scrypt good against offline
cracking makes it an amplifier online: a ~200-byte request causes a 32 MB
allocation. Go serves each request on its own goroutine with no cap, so the
allocations are concurrent.

**What an attacker does.** 100 concurrent signups ⇒ ~3.2 GB of live heap, and
the process is killed. There is no lockout, so this needs no valid account. A
slower variant just enumerates mobile numbers to create rows indefinitely —
`farmers.mobile_number` is `UNIQUE NOT NULL` ([schema.sql:43](schema.sql:43)),
so the row count is bounded by the phone-number space rather than by anything the
application enforces.

**Fix.** Rate-limit `/auth/*` per IP, and bound concurrent password hashing with
a buffered-channel semaphore sized to available memory.

### H6 — Chat session map grows without bound

**What the code does.** `Memory` holds `sessions map[string][]Message`
([memory.go:21-25](internal/chatbot/memory.go:21)). `Append` caps each session at
`CHATBOT_MAX_HISTORY` (default 20) by trimming from the front
([memory.go:49-57](internal/chatbot/memory.go:49)), but nothing ever removes a
*session*. `Clear` runs only when a client calls `/chatbot/reset`
([onboarding_handler.go:402-413](internal/httpapi/onboarding_handler.go:402)),
and the session id comes straight from the request body, defaulting to a fresh
UUID when absent ([onboarding_handler.go:358-361](internal/httpapi/onboarding_handler.go:358)).

**Why it is wrong.** The retention bound is per session, not global. The
comparable cache in this codebase — `ExprCache` — gets both a TTL and a max-size
LRU ([pipeline/cache.go:141-160](internal/pipeline/cache.go:141)); `Memory` got
neither. The code comment says process-local storage is deliberate for parity,
which is fine; the absence of eviction is a separate, unintended gap.

**What an attacker does.** Sends `/chatbot/chat` with a new random `session_id`
each time. Every accepted reply retains up to 20 messages forever. Because the
frontend also mints a fresh `crypto.randomUUID()` per field selection, this also
happens benignly: a long-running server accumulates sessions from ordinary use
and never releases them.

**Fix.** Give `Memory` a TTL and a max-session bound with LRU eviction, reusing
the `ExprCache` approach.

### H7 — `/api/sample` exposes the previous caller's analysis

**What the code does.** `Analyze` stores its result under both a derived cache
key and the process-global sentinel `"__last__"`
([analyze.go:137-138](internal/pipeline/analyze.go:137), [cache.go:22](internal/pipeline/cache.go:22)).
`/api/sample` reads that sentinel with no notion of who ran the analysis
([analyze_handler.go:254](internal/httpapi/analyze_handler.go:254),
[analyze.go:234-254](internal/pipeline/analyze.go:234)) and the route is
unauthenticated.

**Why it is wrong.** This is inherited behaviour, filed as K6, and the *storage*
was correctly modernised into a TTL'd bounded cache. But the sharing semantics
were preserved as-is, so the sentinel is a single global slot: whoever analysed
last owns it, and anyone can read it.

**What an attacker does.** Polls `GET /api/sample?lat=…&lng=…&band=NDVI` and
receives index values sampled from whatever field another user most recently
analysed — including the ability to probe arbitrary coordinates against that
user's cached image, which reveals both the values and, by scanning for non-null
responses, the approximate location of someone else's field. No credentials
required.

**Fix.** Key the sentinel per session or per authenticated farmer. The reason it
is global is that the frontend sends no field identifier on the sample call, so
this needs either a frontend change (currently frozen) or a server-side session
cookie. Note K6 documents the sharing as intentional-for-parity, so this is a
product decision, not a silent bug fix.

---

## Top 10 by ROI

1. **C1 — validate `JWT_SECRET_KEY` at startup.** Ten lines in `Validate()`,
   closes a total authentication bypass. Do this first.
2. **H2 — pass `context.Background()` in `firebase` and `firestore`.** Two
   one-line changes; the fix and its rationale already exist in `gee/session.go`.
   Prevents a recurrence of an outage this project has already had.
3. **H1 — OTP attempt counter.** ~15 lines in `VerifyOTP`; removes an unbounded
   guessing budget.
4. **M6 + M5 — fix the Docker base image and add `.dockerignore`.** Minutes of
   work; one is a build that does not reproducibly work, the other puts `.env`
   and `serviceAccountKey.json` into image layers.
5. **H3 — `MaxBytesReader` on request bodies.** One line in `readBody`, removes
   the cheapest OOM vector.
6. **H4/H5 — per-IP rate limiting middleware on `/api/*`, `/auth/*`, `/chatbot/*`.**
   One middleware, applied in `router.go`; simultaneously caps Earth Engine
   spend, Gemini spend, SMS spend and the scrypt amplifier.
7. **H6 — TTL + LRU on `Memory`.** Copy the `ExprCache` pattern that already
   exists in the same repo.
8. **M11 + L5 — `defer smsSvc.Close()` and delete the three dead symbols.**
   Trivial, and keeps `staticcheck` honest.
9. **M2 — unique constraint on `farmer_locations`.** One migration; prevents
   duplicate rows accumulating silently on every onboarding retry.
10. **M9 — unit tests for `service` and `middleware`.** The two packages with the
    most security-relevant logic both sit at 0%. Start with OTP verification,
    `assertOwnership`, and the JWT middleware's four failure branches.

Deliberately *not* in this list: **M10 (N+1)** and **H7's root cause**, because
both are documented parity decisions (K7, K6) that need a product call rather
than a code fix, and **M4 (migrations)**, which is right but is a project, not a
patch.

---

## What I did not cover

- **`frontend/`** — out of scope by repo policy (`CLAUDE.md`: "Never modify
  `frontend/`"). I read `vite.config.js` and `KrishiMitraPanel.jsx` only to trace
  which backend routes the client actually calls. The frontend's own dependency
  tree, its 21 pre-existing eslint errors, and its XSS surface are unaudited.
- **`internal/gee/eeexpr/`** (~1,100 lines across `api.go`, `node.go`,
  `canonical.go`) — read structurally, not line-by-line. It is pure expression
  construction with no I/O, no user-controlled strings reaching an interpreter,
  and dedicated golden tests. Lower risk per line than the auth and HTTP surface
  I prioritised.
- **The numeric pipeline internals** — `smooth.go`, `indices.go`, `grid.go`,
  `stats.go`, `sentinel1.go`, `radar_*.go`. These are deliberately literal ports
  whose correctness criterion is byte-parity against captured fixtures, and they
  are covered by the numeric-parity layer. Auditing them against mathematical
  intent rather than against the fixtures would contradict the project's own
  correctness definition.
- **`tools/contract/main.go`** (805 lines) — I read its skip-list structure
  (`environmentDependent`, `coldStartOnly`, `sharedStateWrites`, `dateDependent`)
  and confirmed each skip records a reason, but did not audit the comparison
  logic itself. A bug there would weaken the safety net without failing anything.
- **Dependency CVEs** — not checked. `govulncheck` was not run because it needs
  network access to the vulnerability database, and I did not want to make
  outbound calls during a read-only audit. Recommend running
  `govulncheck ./...` separately; the dependency tree is large (80+ indirect
  modules, including a full OpenTelemetry and gRPC stack pulled in by
  `cloud.google.com/go/firestore`).
- **Licence review** — not performed. All direct dependencies are the usual
  BSD/MIT/Apache-2.0 Go ecosystem packages, but I did not enumerate them.
- **Runtime behaviour** — no load testing, no profiling. Every performance
  finding above is derived from code structure (allocation sizes, loop bounds,
  query counts), not from measurement. The 26-second `/api/analyze` figure cited
  in H4 comes from server logs observed earlier in this session, not from a
  benchmark.
- **`schema.sql` as deployed** — I read the checked-in file but did not connect
  to a database, so I cannot confirm the live schema matches it. Given M4 (no
  migration tooling), drift between the two is plausible and worth verifying.

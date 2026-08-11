# REPORT — Project Pragya: Python → Go Backend Migration

Autonomous run against `PRAGYA_GO_MIGRATION_PRD.md`.
Branch: `feat/pragya-go-migration`. Nothing was pushed; no history was rewritten.

**Headline:** the analytics core is done and provably correct — the Go backend
returns byte-identical results to the Python one for every Sentinel-2 and
Sentinel-1 endpoint, verified live on the same day against the same polygons.
The data layer (Postgres, Firestore, SMS, onboarding) is **not** done; those
endpoints validate their input correctly and then answer 501. Details in §4.

---

## 1. Summary — what got built, phase by phase

### Phase 0 — Groundwork ✅

Go module (`go 1.23`), `Makefile` with a `verify` gate, `.golangci.yml`,
`staticcheck.conf`, multi-stage distroless `Dockerfile`. `backend/` moved to
`legacy-python/` (57 tracked renames); the dead interactive CLI under
`backend/legacy/` deleted after confirming nothing imports it.

`internal/config` is a 1:1 port of `config.py` + `chatbot/config.py` + the
palettes from `app.py`, with thresholds stored as **descending ordered slices**
because a Go map has no order and the Python sorts its dict keys descending at
every call site. `config_test.go` is a second, independent transcription of PRD
Appendix A, so a typo would have to be made identically twice.

**Every golden was captured from the live Python backend before anything moved:**

| Layer | What | Count |
|---|---|---:|
| 1 | EE expression graphs (§5.7 G1–G24, plus arithmetic variants) | 44 |
| 1 | Live `maps` request + tile-URL template | 1 |
| 2 | HTTP contract cases (§12.2) | 73 |
| 3 | Numeric parity, 5 polygons × S2 + S1 (§12.3) | 5 |
| — | Werkzeug password hashes (§8.2) | 9 |
| — | Chatbot system-prompt renders (§10.11) | 2 |

### Phase 1 — HTTP skeleton ✅

All 24 routes of §10.1, global CORS, a `slog` handler reproducing the Python log
format (em dash and all), `ValidatePolygon` as a byte-exact port of §6.1, the
three distinct error envelope shapes, marshmallow-compatible validation, and
JWT verification. `cmd/server` runs its startup probes **once in a goroutine**
rather than reproducing Flask's per-request `@app.before_request` hack, with a
300 s write timeout and a 30 s graceful drain.

### Phase 2 — Earth Engine core ✅ (the critical path)

`internal/gee/eeexpr` builds Earth Engine expression-graph JSON by hand: node
types, a compiler that interns shared subtrees and hoists function-definition
bodies, and a canonicaliser that compares graphs **semantically** (fully
inlined) and **up to alpha-equivalence**, so reference-key ordering and
mapping-variable names — both of which depend on Python build order — cannot
cause false failures.

All 24 computations of §5.7 are implemented and gated by **strict equality**
against fixtures captured from the real Python client.

`internal/gee` adds the session (service account primary, `earthengine
authenticate` refresh token as a developer fallback), the REST client
(`value:compute`, `table:computeFeatures` with `nextPageToken` paging,
`maps`), and the §13.4 retry policy — 429 and 5xx only, with backoff, jitter,
and immediate abort on context cancellation.

**Phase 2 exit gate, live:** `s2_scene_count = 17` and `grid cells = 266`, both
matching the Python client exactly.

### Phase 3 + 4 — Sentinel-2 and Sentinel-1 pipelines ✅

`internal/pipeline` orchestrates scene counting, grid auto-coarsening, per-cell
reduction, Gaussian smoothing, statistics and tile generation, behind an
`EEClient` interface so the package never touches HTTP directly (§3.2).

`smooth.go` is an exact port of `_smooth_grid_values` including the quirks that
are load-bearing for parity: the centroid counts the duplicated closing vertex,
`j == i` is included in the weighted sum, iteration runs in slice order because
float addition is not associative, and smoothing operates on the
already-rounded 4 dp values.

`cache.go` replaces `app._last_indexed_image` with a TTL'd, bounded, LRU-evicting
cache (objective O4) while preserving the shared `"__last__"` semantics the
frontend depends on (K6).

### Phase 5 — Data layer ⚠️ PARTIAL

Only `internal/crypto` landed. See §4.

### Phase 6 — Chatbot ✅

`prompt_body.go` is **generated** from the Python f-string by
`legacy-python/tools/gen_prompt.py` rather than retyped, so no character can
drift; it renders identically to the Python for both fixture cases. Plus
mutex-guarded session memory and a direct Ollama `/api/chat` client — LangChain
dropped entirely (O5).

### Phase 7 — Cutover ⚠️ PARTIAL

Endpoint verification done and recorded. `legacy-python/` deliberately **not**
deleted — see §4.1.

---

## 2. Endpoint table

Full detail with actual response bodies:
[docs/ENDPOINT_VERIFICATION.md](docs/ENDPOINT_VERIFICATION.md) — **61 cases,
61 passing**, every one a real HTTP request against the running Go server.

| Endpoint | Method | Test cases run | Result | Notes |
|---|---|---|---|---|
| `/health` | GET | happy | **PASS** | 4 keys exactly; capability flags track the atomics |
| `/api/analyze` | POST | happy, 7 invalid-input branches, no-imagery, water body, large field (auto-coarsening), tiny plot | **PASS** | 266 cells, byte-identical to Python |
| `/api/analyze-dates` | POST | happy, missing geometry, invalid polygon | **PASS** | ascending distinct dates |
| `/api/analyze-day` | POST | missing date, missing geometry, invalid polygon, no imagery | **PASS** | 200-with-error preserved |
| `/api/analyze-radar-dates` | POST | happy, missing geometry, invalid polygon | **PASS** | |
| `/api/analyze-radar` | POST | happy, missing geometry, no-imagery ×2 wordings | **PASS** | both halves of the inline conditional |
| `/api/sample` | GET | cold cache 404, happy, lower-cased band, bad coords, bad band | **PASS** | check ordering verified |
| `/api/auth/verify-token` | POST | missing idToken, invalid token | **PARTIAL** | 400 branch correct; verification is 501 |
| `/api/auth/send-otp` | POST | too short, non-digit | **PARTIAL** | happy path not run — would send a real billed SMS |
| `/api/auth/verify-otp` | POST | missing fields, wrong OTP | **PARTIAL** | no OTP store yet |
| `/auth/signup` | POST | bad mobile (422), short password (422) | **PARTIAL** | validation correct; service is 501 |
| `/auth/login` | POST | bad mobile (422), empty password (422) | **PARTIAL** | as above |
| JWT middleware | — | no header, non-Bearer, bad signature, expired | **PASS** | all four statuses and messages exact |
| `/farmer/basic-details` | POST | invalid body, unauthorized | **PARTIAL** | validation + auth correct; service is 501 |
| `/farmer/location` | POST | bad pin | **PARTIAL** | as above |
| `/farmer/pincode/:pin` | GET | lookup | **NOT IMPLEMENTED** | 501 |
| `/farm`, `/crop`, `/irrigation`, `/soil`, `/consent` | POST | invalid body each | **PARTIAL** | validation + auth correct; services are 501 |
| `/dashboard` | GET | authorized | **NOT IMPLEMENTED** | 501 |
| `/chatbot/chat` | POST | empty message | **PARTIAL** | happy path needs a running Ollama |
| `/chatbot/reset` | POST | missing session_id, happy | **PASS** | |
| `/chatbot/health` | GET | happy | **PASS** | |

### Numeric parity, measured live

Go and Python queried on the same day with the same polygon:

| Metric | Result |
|---|---|
| Grid cells | 266 vs 266 |
| Per-cell values, 266 cells × 7 bands | **max delta 0.0** |
| Farm-summary means, 7 bands | **max delta 0.0** |
| NDVI histogram, 20 buckets | **max delta 0.0** |
| Interpretation labels | 0 mismatches |
| `confidence` | 0.887 vs 0.887 |
| Nulls | 0 vs 0 — no null ever became 0 (§13.1) |

`go vet`, `staticcheck`, and `go test ./... -race` are clean across all
packages.

---

## 3. Auto-fixed issues

Small errors found and fixed without stopping:

1. `validator/v10` splits tag strings on commas before a custom validation sees the param, so `mlen=1,255` parsed as `mlen=1` plus an undefined tag `255` and panicked at struct-cache build time → 500 on every schema route. Switched custom tags to a `:` separator.
2. Required schema fields had to become **pointers**: marshmallow distinguishes "key absent" (`Missing data for required field.`) from "present but empty" (`Length must be between 1 and 255.`), which a plain Go string cannot express.
3. PRD §6.1's `strconv.FormatFloat(v,'g',-1,64)` prints `181` where Python prints `181.0`; also the int-vs-float distinction depends on the JSON *literal*, needing `json.Number`.
4. Same `json.Number` issue in the chatbot prompt (`30` vs `30.0`).
5. `ImageCollection.select` does not exist — `.select` on a collection is `.map(img => img.select(...))`.
6. `copyProperties` resolves to `Image.copyProperties`, not `Element.copyProperties`.
7. `feature.set(dict)` is `Element.setMulti{object, properties}`, not `Element.set`.
8. `Image.reproject` takes a `Projection` node for `crs`, not a bare string.
9. `addBands([list])` combines the list into one image first and recurses on `srcImg`, not `dstImg`.
10. `X-Goog-User-Project` is **required** with user credentials, else EE answers 403 "Not signed up for Earth Engine" despite the project being registered.
11. `ee/oauth.py` declares `CLIENT_ID` as a parenthesised multi-line string concatenation; a naive regex silently returned a truncated value.
12. Cell geometries must carry `geodesic` and `crs`, which the Python client attaches on `getInfo` but `table:computeFeatures` omits.
13. `config.Load` and the oauth.py lookup now walk up to the repo root, so tests and a binary started from any directory find `.env`.
14. `frontend/node_modules` vendors a Go package that `./...` picked up; scoped the package list instead of touching `frontend/` (forbidden by §0.2).
15. Removed a hardcoded OAuth client secret I had initially written from memory — it is now read at runtime from env or the installed `ee/oauth.py`, and never committed.
16. Assorted lint: `ST1000` package comments, `ST1023`, unused Phase-3 helpers, a deprecated `google.CredentialsFromJSON` documented rather than blind-swapped, and `ST1005` disabled repo-wide with a written justification (the error strings *are* the frozen contract).

---

## 4. UNRESOLVED / COMPLEX ISSUES

### 4.1 Phase 5 (data layer) is not implemented — the main gap

**What:** `internal/db`, `internal/repo/*` (7 repositories), `internal/service/*`
(onboarding), `internal/firebase`, `internal/firestore`, `internal/service/sms.go`,
`internal/service/pincode.go`, and `internal/jwtutil` (the token *issue* path)
do not exist. E8–E21 validate correctly and then answer 501.

**Where:** `internal/httpapi/onboarding_handler.go` — every `s.notImplemented(...)` call.

**Why it is not trivial:** it is roughly 900 LOC of Python across 20 files, and
the acceptance bar (§14 Phase 5) is behavioural — "an existing production user
logs in and loads `/dashboard`" — which needs the repositories, the type
coercions of §10.10 (`NUMERIC(10,2)` → `2.5` not `"2.50"`, `DATE` → `YYYY-MM-DD`
not RFC3339, `ST_AsGeoJSON` → raw JSON, `latest_vi_report` → null), Firestore
session mirroring, and the N+1 crops query preserved deliberately (K7). Half of
it would be worse than none: a partly-working auth path that writes real rows
into the farmers table is exactly the kind of thing that is hard to unwind.

**What was done instead:** the genuinely blocking, hard-to-get-right piece —
Werkzeug hash compatibility — is complete and verified **bidirectionally**
against the real Python environment. Go verifies every Python-generated hash
(including the `scrypt:32768:8:1` format all 2 production rows use), and
Python's `check_password_hash` accepts Go-generated hashes, so a rollback is
safe. That de-risks the rest of Phase 5 considerably.

**Options:**
- **(a) Finish Phase 5 as specified.** ~2–3 days. Highest fidelity; the PRD's §10.10 type notes are detailed enough to follow directly. **Recommended.**
- **(b) Run a hybrid cutover** — Go serves `/api/*` and `/chatbot/*`, Python keeps `/auth`, `/farmer`, `/farm`, `/dashboard`, behind one reverse proxy. Ships the finished 80% now; costs a proxy and two runtimes.
- **(c) Keep Python entirely until Phase 5 lands.** Zero risk, zero benefit until then.

**Recommendation: (a)**, with **(b)** as the fallback if the analytics work needs to ship before the onboarding flow is ready.

### 4.2 No Earth Engine service account exists (objective O7 unmet)

**What:** the production auth path requires a GCP service account registered for
Earth Engine. None exists; root `serviceAccountKey.json` is Firebase, not EE.

**Where:** `internal/gee/session.go`, `NewSession` — the primary branch is dead
in this environment and everything falls through to the developer path.

**Why it is not trivial:** it is a console + billing task requiring GCP project
permissions, not a code change, and registration at
`signup.earthengine.google.com/#!/service_accounts` can take time to propagate.

**What was done:** both paths are implemented. The developer fallback works and
all live tests pass through it. The refresh-token path is explicitly marked
developer-only in the code.

**Options:** (a) provision the service account before deploy — **recommended**,
it is the only path that satisfies O7; (b) deploy on a VM with the developer
credentials mounted — works, but startup then depends on a token a human
created, which is exactly what O7 forbids; (c) use Application Default
Credentials on Cloud Run with the runtime service account registered for EE —
clean if the deployment target is Cloud Run.

### 4.3 K11 — the pincode User-Agent block (owner decision applied)

`api.postalpincode.in` TCP-resets the default `python-requests` User-Agent, so
`GET /farmer/pincode/<pin>` currently returns **500 always**, and
`POST /farmer/location` silently stores empty state/district/taluka. You
accepted the fix as the new baseline, so the Go client will set an explicit
User-Agent and return real 200/404 branches. The two `e15_*` goldens captured
the broken 500 and are excluded from the parity gate with the reason recorded.
**This is the one sanctioned deviation from §0.8.**

### 4.4 K13 — the confidence score's cloud term is dead

`aggregate_mean("CLOUDY_PIXEL_PERCENTAGE")` returns **null**, because the two
`.map()` calls strip image properties before it runs. Python's `avg_cloud or 0`
turns that into `0`, so the cloud component contributes a constant `0.30` to
every confidence score on `/api/analyze`. Verified against the running Python
backend; **reproduced exactly, not fixed** (§0.8). This was found only because
Go and Python were diffed live — it is invisible from the code alone.

### 4.5 The Layer-3 numeric goldens drift

The 90-day lookback window is relative to *today*, so goldens captured on one
day cannot be compared against a run on another. `tools/verify_endpoints.py`
sidesteps this by querying both backends live in the same session. The committed
`testdata/golden/numeric/` files are useful as shape references but are **not**
usable as numeric assertions after the capture date.

**Options:** (a) pin `LOOKBACK_DAYS`' end date via an env override during
capture — **recommended**, makes goldens durable; (b) keep using live
side-by-side comparison, which is what works today but needs the Python backend
alive; (c) drop the numeric goldens and rely on the live runner.

### 4.6 Two endpoints could not be exercised

- `POST /api/auth/send-otp` happy path — would send a real, billed SMS through
  the nationalbulksms gateway. Only the 400 branches were run.
- `POST /chatbot/chat` happy path — needs a running Ollama server with the model
  pulled. The prompt render, memory and client are unit-tested; the round trip
  is not.

---

## 5. Decisions & assumptions

1. **Worked on `feat/pragya-go-migration`**, not `main`, and committed after each phase. You had earlier said you would create a branch; this run's instructions said to commit per task. A branch satisfies both.
2. **Fixtures beat the PRD.** Wherever §5.3/§5.6/§8.3/§6.1 disagreed with what the real Python client emits, the captured fixture won. All twelve corrections are in `docs/KNOWN_ISSUES.md`.
3. **Graph comparison is semantic, not textual** — inlined and alpha-equivalent. Reference-key ordering and `_MAPPING_VAR_<n>_<i>` names are artefacts of Python build order and carry no meaning to Earth Engine.
4. **Added `g07_indexed_arith` and six `*_arith` fixtures.** The originals embed `Image.parseExpression`, which §7.3 replaces with band arithmetic, so they could never match. Regenerating them from an equivalent arithmetic Python expression converted six loose shape-checks into strict equality assertions.
5. **`gee_ready`/`firebase_ready` are capability-compared, not value-compared,** in the contract runner. They describe the runtime state of whichever process answers; comparing them across two processes is meaningless. `TestHealth` asserts each flag genuinely tracks its atomic.
6. **`no_imagery.json` added** (87°N, above Sentinel-2's orbital coverage). The PRD's `cloudy_region.json` was supposed to yield zero scenes but mid-Pacific has S2 coverage and returns a normal success body.
7. **The `n < 2` smoothing early-return is covered by a Go unit test, not a fixture polygon.** `coveringGrid` snaps to a grid, so `tiny_plot.json` yields 6 cells, not fewer than 2.
8. **`ST1005` disabled repo-wide** with a written justification: the error strings are user-facing frozen contract text, and rewording them to satisfy the linter would fail the contract suite.
9. **The OAuth client secret is never committed.** Read at runtime from env or the installed `ee/oauth.py`.
10. **`legacy-python/` was NOT deleted.** §0.7 keeps it until Phase 7 sign-off; Phase 5 is incomplete, so it is still the only working implementation of E11–E21.
11. **`CLAUDE.md` got a stale-state banner** rather than the full §16 rewrite, which belongs at real cutover. `docs/CHANGELOG.md` and `docs/KNOWN_ISSUES.md` carry the current state.
12. **The contract runner treats 501 and "GEE not wired" 503s as `STUB`, not `FAIL`,** so each phase's gate measures only what that phase claims to deliver.

---

## 6. What you need to do next

1. **Review the branch.** `git log --oneline main..feat/pragya-go-migration` — six commits, each with its own verification.
2. **Provision the Earth Engine service account** (§4.2). It is a console task only you can do, and it blocks objective O7 and any deploy.
3. **Decide Phase 5: option (a) finish it, or option (b) hybrid cutover** (§4.1). This is the only thing between here and a full replacement.
4. **Confirm the K11 pincode decision is still what you want** now that you can see it in context (§4.3) — it is the one place the Go backend deliberately behaves differently from Python.
5. **Run the frontend against the Go backend** for the §12.4 visual check: draw a polygon, analyse, toggle all seven layers, switch dates, hover, then the radar layers. The JSON is proven identical, but tile *rendering* is the one thing a JSON diff cannot catch.
   ```bash
   go build -o bin/server ./cmd/server && ./bin/server -port 5000
   ```
6. **Start Ollama and exercise the chatbot round trip** (§4.6) — the only untested path in an otherwise complete Phase 6.
   ```bash
   ollama serve
   ```
7. **Before deleting `legacy-python/`,** re-run `legacy-python/tools/exercise_endpoints.py` and confirm 61/61 still pass with Phase 5 complete. Delete it in its own commit, as §14 requires.

# Known Issues — preserved during the Go migration

PRD reference: `PRAGYA_GO_MIGRATION_PRD.md` §13.9.

These are **deliberately reproduced** by the Go port. Rule §0.8 forbids bundling
behavioural fixes into the migration: the frontend and the contract test suite
both depend on current behaviour, so each of these is ported as-is and fixed
afterwards in a separate, clearly-scoped commit (PRD §14 Phase 8).

Do not "fix" any of these while porting. If a Go response differs from the
Python one because of an issue below, the Go response is wrong.

| ID | Issue | Where (original Python) | Go handling |
|---|---|---|---|
| K1 | `confidence` is 0–1 in the API but documented 0–100 in `schema.sql` / README ("56.46%") | `stats_service.compute_confidence` | Emit 0–1. The frontend does the formatting. |
| K2 | `CVI_WEIGHTS` in `config.py` (0.70/0.10/0.05/0.10/0.05) disagree with README and the `vi_reports` comment (0.35/0.25/0.15/0.15/0.10) | `config.py` vs docs | `internal/config` uses the **config.py** values. Docs corrected in Phase 7. |
| K3 | `_interpret_cvi` docstring says 0.6/0.3; the config says 0.5/0.25 | `grid_service` | Config wins. The code was always right; only the docstring lied. |
| K4 | `.map(img => img.divide(10000))` also divides the SCL band, not just the reflectance bands | `gee_service` | Reproduced. The SCL mask is applied *before* the divide, so results are unaffected — but the graph shape must match. |
| K5 | SMS template text says **"LaundryLy"** — wrong brand, but the text is DLT-registered in India | `sms_service.send_otp` | Ported verbatim with a `// KNOWN-ISSUE` comment. Changing it silently breaks delivery; it needs re-registration, which is a product decision. |
| K6 | `/api/sample` reads a process-global last image, shared across all users and lost on restart | `app.py` | Structurally replaced by the TTL'd `ExprCache` (§7.7), but the shared `"__last__"` sentinel semantics are preserved because the frontend sends no field identifier. |
| K7 | `GET /dashboard` runs one crops query per farm (N+1) | `dashboard.py` | Preserved for parity. Batching it is a Phase 8 change. |
| K8 | OTP store never evicts expired entries (unbounded growth) | `sms_service` | Go adds a janitor goroutine. Expiry *semantics* are identical, so this is invisible to the contract. |
| K9 | `vite.config.js` proxies `/api` only; `CLAUDE.md` documents `/chatbot` as proxied but it is not | frontend vs docs | Frontend is not modified (§0.2). The axios client reaches `/chatbot` via direct CORS, which is why CORS must be applied globally (§9.3). |
| K10 | `app._last_radar_image` is written but never read | `app.py` | The write is preserved under the `"__last_radar__"` sentinel with a `// parity: unused by any handler` comment. |

## Found during Phase 0 (not in the PRD)

| ID | Issue | Where | Notes |
|---|---|---|---|
| K11 | `GET /farmer/pincode/<pin>` **always returns 500**. `api.postalpincode.in` rejects the default `python-requests/x.y.z` User-Agent with a TCP reset; any other UA works. Consequence: `POST /farmer/location` always stores empty `state`/`district`/`taluka` via the §7.10 "on any failure, continue" path. | `utils/pincode.py` (no UA header set) | **RESOLVED — fix accepted as the new baseline** (owner decision, Phase 0). Go sets an explicit non-blocked User-Agent and returns the real 200 / 404 / 500 branches per §10.10. This is the one sanctioned deviation from §0.8. The `e15_pincode_success` and `e15_pincode_notfound` goldens capture the *broken* 500 and are therefore superseded: the contract runner asserts them against the §10.10 spec instead of the golden, and they are excluded from the byte-parity gate. `POST /farmer/location` will now store real state/district/taluka, which is a data-shape change the frontend already tolerates (the fields were simply empty before). |
| K12 | `services/auth.py` mints tokens whose `sub` is the farmer UUID as a string, and `flask_jwt_extended` returns **401** (not 422) with `Missing 'Bearer' type in 'Authorization' header. Expected 'Authorization: Bearer <JWT>'` for a non-Bearer header. PRD §8.3's table claims 422 with different wording. | `middlewares/auth.py` via flask-jwt-extended 4.7.4 | Verified against the running app; goldens `jwt_*.json` are authoritative. The Go middleware must emit 401 here. |

## Verified PRD corrections

Captured empirically in Phase 0; the fixtures under `internal/gee/eeexpr/testdata/`
are authoritative over the PRD prose.

| PRD section | Claim | Reality |
|---|---|---|
| §5.3 | `Collection.filterDate{collection,start,end}` | `Collection.filter` + `Filter.dateRangeContains{leftValue: DateRange{start,end}, rightField:"system:time_start"}` |
| §5.3 | — | `filterBounds` → `Collection.filter` + `Filter.intersects{leftField:".all", rightValue: Feature{geometry}}` |
| §5.3 | — | `ee.Filter.lt` → `Filter.lessThan{leftField, rightValue}` |
| §11.1 | `Image.rename` takes `bandNames` | it takes `input`, **`names`** |
| §11.1 | `Image.divide(scalar)` | `Image.divide{image1, image2}`; the scalar is promoted to `Image.constant{value}` |
| §5.3 | `.median()` | `reduce.median{collection}` |
| §5.6 | `maps` body carries `visualizationOptions` + `bandIds:["NDVI"]` | body is `{expression, fileFormat:"AUTO_JPEG_PNG", bandIds:[]}`; vis params are folded into the **expression** as `Image.visualize{image,min,max,palette}` |
| §5.6 | palette hex must have `#` stripped | the `#` is **kept** — stripping it would break every tile |
| §5.5 | `~/.config/earthengine/credentials` holds `client_id`/`client_secret` | it holds only `refresh_token`, `redirect_uri`, `scopes`; the client id/secret are library constants in `ee.oauth` |
| §5.8 | `Collection.map` arg key unverified | confirmed `baseAlgorithm`, with `functionDefinitionValue{argumentNames:["_MAPPING_VAR_0_0"], body:"<ref string>"}` |
| §12.3 | `cloudy_region.json` yields zero clean scenes | mid-Pacific **has** S2 coverage and returns a success body. The real empty-collection case needs a polygon above ~84°N (`no_imagery.json`). |
| §12.3 | `tiny_plot.json` gives fewer than 2 grid cells | it gives 6. `coveringGrid` snaps to a grid, so the `n < 2` smoothing early-return is covered by a Go unit test instead. |

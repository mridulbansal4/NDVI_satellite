"""contract_corpus.py — the frozen HTTP contract request corpus.

PRD reference: PRAGYA_GO_MIGRATION_PRD.md §12.2.

One case per endpoint per outcome branch in §10. This module is the single
definition of the corpus; `dump_contract.py` fires it at the Python backend to
produce goldens, and the Go runner (tools/contract) replays the identical list
against the Go backend and diffs.

Each case:
    id       — stable slug, also the golden filename
    method   — GET | POST
    path     — request path
    body     — JSON body (POST) or None
    query    — query-string dict (GET) or None
    auth     — None | "valid" | "expired" | "badsig" | "malformed"
    expect   — documented expected status, for a fast eyeball check
    phase    — "setup" cases run first and are not diffed
    note     — why the case exists / what branch it pins

NOT CAPTURED, deliberately:
  * POST /api/auth/send-otp with a valid number. It hits a paid SMS gateway and
    would send a real message. Only the 400 branch is captured. The 200/502
    branches must be verified manually against a staging gateway.
  * The 503 "GEE not initialised" branches. GEE is ready on this machine, so
    they cannot be produced here; they are captured in Phase 1 by starting the
    Go server with GEE_PROJECT_ID unset and asserting the two distinct wordings
    from §10.4 directly.
"""

import json
import pathlib

REPO = pathlib.Path(__file__).resolve().parents[2]
POLY = REPO / "testdata" / "polygons"


def polygon(name: str) -> dict:
    return json.loads((POLY / f"{name}.json").read_text())


# A mobile number reserved for contract testing. The setup phase deletes and
# recreates this farmer so signup/login/onboarding branches are deterministic.
TEST_MOBILE = "9000000001"
TEST_PASSWORD = "contract-test-pw"

NASHIK = polygon("nashik_vineyard")
TINY = polygon("tiny_plot")
CLOUDY = polygon("cloudy_region")
# Above Sentinel-2's orbital coverage limit (~84N), so the S2 collection is
# reliably empty. cloudy_region does NOT work for this: mid-Pacific has S2
# coverage and returns a normal success body.
NO_IMAGERY = polygon("no_imagery")


def _c(cid, method, path, *, body=None, query=None, auth=None, expect=200,
       phase="main", note=""):
    return {
        "id": cid, "method": method, "path": path, "body": body, "query": query,
        "auth": auth, "expect": expect, "phase": phase, "note": note,
    }


CASES = [
    # ── E1 health ────────────────────────────────────────────────────────────
    _c("e1_health", "GET", "/health", note="§10.2 always 200"),

    # ── E7 sample: 404 branch MUST run before any analyze populates the cache
    _c("e7_sample_no_analysis", "GET", "/api/sample",
       query={"lat": "20.0116", "lng": "73.7908"}, expect=404,
       note="§10.7 no cached image yet — ordering-sensitive, keep first"),

    # ── E2 analyze ───────────────────────────────────────────────────────────
    _c("e2_analyze_no_body", "POST", "/api/analyze", body={}, expect=400,
       note="§10.3 missing geometry — long wording"),
    _c("e2_analyze_not_object", "POST", "/api/analyze",
       body={"geometry": "not-an-object"}, expect=400, note="§6.1 check 1"),
    _c("e2_analyze_wrong_type", "POST", "/api/analyze",
       body={"geometry": {"type": "MultiPolygon", "coordinates": []}},
       expect=400, note="§6.1 check 2 — MultiPolygon is rejected"),
    _c("e2_analyze_empty_coords", "POST", "/api/analyze",
       body={"geometry": {"type": "Polygon", "coordinates": []}}, expect=400,
       note="§6.1 check 3"),
    _c("e2_analyze_short_ring", "POST", "/api/analyze",
       body={"geometry": {"type": "Polygon",
                          "coordinates": [[[0, 0], [1, 0], [0, 0]]]}},
       expect=400, note="§6.1 check 4"),
    _c("e2_analyze_bad_point", "POST", "/api/analyze",
       body={"geometry": {"type": "Polygon",
                          "coordinates": [[[0, 0], [1], [1, 1], [0, 0]]]}},
       expect=400, note="§6.1 check 5"),
    _c("e2_analyze_bad_lon", "POST", "/api/analyze",
       body={"geometry": {"type": "Polygon",
                          "coordinates": [[[181.0, 0], [1, 0], [1, 1], [181.0, 0]]]}},
       expect=400, note="§6.1 check 6 — number formatting matters"),
    _c("e2_analyze_bad_lat", "POST", "/api/analyze",
       body={"geometry": {"type": "Polygon",
                          "coordinates": [[[0, 91.5], [1, 0], [1, 1], [0, 91.5]]]}},
       expect=400, note="§6.1 check 7"),
    _c("e2_analyze_no_imagery_true", "POST", "/api/analyze",
       body={"geometry": NO_IMAGERY}, expect=200,
       note="§13.2 the real 200-with-error branch — S2 has no scenes above ~84N"),
    _c("e2_analyze_no_imagery", "POST", "/api/analyze", body={"geometry": CLOUDY},
       expect=200,
       note="mid-Pacific: DOES have S2 coverage, so this is a success body. "
            "Kept as a non-Indian-latitude success case."),
    _c("e2_analyze_success", "POST", "/api/analyze", body={"geometry": NASHIK},
       expect=200, note="§10.3 full success body"),
    _c("e2_analyze_tiny", "POST", "/api/analyze", body={"geometry": TINY},
       expect=200, note="§12.3 exercises the n<2 smoothing early-return"),

    # ── E7 sample: now that /api/analyze has run, the cache is warm ──────────
    _c("e7_sample_success", "GET", "/api/sample",
       query={"lat": "20.0116", "lng": "73.7908"}, expect=200,
       note="§10.7 default band NDVI"),
    _c("e7_sample_band_lowercase", "GET", "/api/sample",
       query={"lat": "20.0116", "lng": "73.7908", "band": "cvi"}, expect=200,
       note="§6.2 band is upper-cased before validation"),
    _c("e7_sample_bad_coords", "GET", "/api/sample", query={"lat": "abc"},
       expect=400, note="§10.7 coord parse failure"),
    _c("e7_sample_bad_band", "GET", "/api/sample",
       query={"lat": "20.0116", "lng": "73.7908", "band": "NOPE"}, expect=400,
       note="§6.2 error text embeds the Python list repr"),

    # ── E3 analyze-dates ─────────────────────────────────────────────────────
    _c("e3_dates_no_body", "POST", "/api/analyze-dates", body={}, expect=400,
       note="§10.4 short 'Missing geometry' wording"),
    _c("e3_dates_invalid", "POST", "/api/analyze-dates",
       body={"geometry": {"type": "Point", "coordinates": [0, 0]}}, expect=400),
    _c("e3_dates_success", "POST", "/api/analyze-dates", body={"geometry": NASHIK},
       expect=200, note="§10.4 ascending distinct YYYY-MM-DD"),

    # ── E4 analyze-day ───────────────────────────────────────────────────────
    _c("e4_day_missing_date", "POST", "/api/analyze-day",
       body={"geometry": NASHIK}, expect=400,
       note="§10.5 'Missing geometry or date'"),
    _c("e4_day_missing_geometry", "POST", "/api/analyze-day",
       body={"date": "2025-02-14"}, expect=400),
    _c("e4_day_invalid_polygon", "POST", "/api/analyze-day",
       body={"geometry": {"type": "Point", "coordinates": [0, 0]},
             "date": "2025-02-14"}, expect=400),
    _c("e4_day_no_imagery", "POST", "/api/analyze-day",
       body={"geometry": NASHIK, "date": "1999-01-01"}, expect=200,
       note="§10.5 200-with-error, message embeds the date"),

    # ── E5 radar dates ───────────────────────────────────────────────────────
    _c("e5_radar_dates_no_body", "POST", "/api/analyze-radar-dates", body={},
       expect=400),
    _c("e5_radar_dates_invalid", "POST", "/api/analyze-radar-dates",
       body={"geometry": {"type": "Point", "coordinates": [0, 0]}}, expect=400),
    _c("e5_radar_dates_success", "POST", "/api/analyze-radar-dates",
       body={"geometry": NASHIK}, expect=200),

    # ── E6 radar ─────────────────────────────────────────────────────────────
    _c("e6_radar_no_body", "POST", "/api/analyze-radar", body={}, expect=400),
    _c("e6_radar_invalid", "POST", "/api/analyze-radar",
       body={"geometry": {"type": "Point", "coordinates": [0, 0]}}, expect=400),
    _c("e6_radar_no_imagery_nodate", "POST", "/api/analyze-radar",
       body={"geometry": CLOUDY}, expect=200,
       note="§10.6 'for this area.' wording"),
    _c("e6_radar_no_imagery_withdate", "POST", "/api/analyze-radar",
       body={"geometry": CLOUDY, "date": "2025-02-14"}, expect=200,
       note="§10.6 'for <date>.' wording — the other half of the conditional"),
    _c("e6_radar_success", "POST", "/api/analyze-radar",
       body={"geometry": NASHIK}, expect=200,
       note="§10.6 date echoes null when omitted"),

    # ── E8 verify-token ──────────────────────────────────────────────────────
    _c("e8_verify_token_missing", "POST", "/api/auth/verify-token", body={},
       expect=400),
    _c("e8_verify_token_invalid", "POST", "/api/auth/verify-token",
       body={"idToken": "not-a-real-token"}, expect=401,
       note="§8.4 — firebase_ready is false in dev; pins that path too"),

    # ── E9 send-otp — 400 branch ONLY. See module docstring. ─────────────────
    _c("e9_send_otp_bad_phone", "POST", "/api/auth/send-otp",
       body={"phone": "12345"}, expect=400),
    _c("e9_send_otp_nondigit", "POST", "/api/auth/send-otp",
       body={"phone": "98765abcde"}, expect=400),

    # ── E10 verify-otp ───────────────────────────────────────────────────────
    _c("e10_verify_otp_missing", "POST", "/api/auth/verify-otp", body={},
       expect=400),
    _c("e10_verify_otp_wrong", "POST", "/api/auth/verify-otp",
       body={"phone": TEST_MOBILE, "otp": "000000"}, expect=401),

    # ── E11 signup ───────────────────────────────────────────────────────────
    _c("e11_signup_bad_mobile", "POST", "/auth/signup",
       body={"mobile_number": "123", "password": "secret1"}, expect=422,
       note="§6.2 — 422, not 400. Inconsistency is preserved."),
    _c("e11_signup_short_password", "POST", "/auth/signup",
       body={"mobile_number": TEST_MOBILE, "password": "abc"}, expect=422),
    _c("e11_signup_success", "POST", "/auth/signup",
       body={"mobile_number": TEST_MOBILE, "password": TEST_PASSWORD},
       expect=201, phase="setup",
       note="creates the contract-test farmer; teardown removes it"),
    _c("e11_signup_duplicate", "POST", "/auth/signup",
       body={"mobile_number": TEST_MOBILE, "password": TEST_PASSWORD},
       expect=409),

    # ── E12 login ────────────────────────────────────────────────────────────
    _c("e12_login_bad_mobile", "POST", "/auth/login",
       body={"mobile_number": "abc", "password": "x"}, expect=422),
    _c("e12_login_no_account", "POST", "/auth/login",
       body={"mobile_number": "9000000099", "password": TEST_PASSWORD},
       expect=401),
    _c("e12_login_wrong_password", "POST", "/auth/login",
       body={"mobile_number": TEST_MOBILE, "password": "definitely-wrong"},
       expect=401),
    _c("e12_login_success", "POST", "/auth/login",
       body={"mobile_number": TEST_MOBILE, "password": TEST_PASSWORD},
       expect=200, phase="setup", note="mints the JWT used by the auth cases"),

    # ── JWT middleware branches (§8.3) — pinned on one protected route ───────
    _c("jwt_missing_header", "GET", "/dashboard", expect=401,
       note="§8.3 {'msg':'Missing Authorization Header'}"),
    _c("jwt_malformed_header", "GET", "/dashboard", auth="malformed", expect=422),
    _c("jwt_bad_signature", "GET", "/dashboard", auth="badsig", expect=422),
    _c("jwt_expired", "GET", "/dashboard", auth="expired", expect=401),

    # ── E13/E14 farmer ───────────────────────────────────────────────────────
    _c("e13_basic_details_invalid", "POST", "/farmer/basic-details",
       body={"name": "", "preferred_language": "klingon"}, auth="valid",
       expect=400, note="§7.11 — pins the marshmallow OneOf wording"),
    _c("e13_basic_details_success", "POST", "/farmer/basic-details",
       body={"name": "Contract Test", "age": 40, "gender": "male",
             "preferred_language": "marathi"}, auth="valid", expect=200),
    _c("e14_location_bad_pin", "POST", "/farmer/location",
       body={"pin_code": "12", "village_name": "Testville"}, auth="valid",
       expect=400),
    _c("e14_location_success", "POST", "/farmer/location",
       body={"pin_code": "422001", "village_name": "Testville"}, auth="valid",
       expect=201),
    _c("e15_pincode_success", "GET", "/farmer/pincode/422001", expect=200),
    _c("e15_pincode_notfound", "GET", "/farmer/pincode/000000", expect=404),

    # ── E16 farm ─────────────────────────────────────────────────────────────
    _c("e16_farm_invalid", "POST", "/farm",
       body={"farm_name": "", "total_area": 0, "area_unit": "bushels",
             "land_ownership": "nope", "latitude": 200, "longitude": 400},
       auth="valid", expect=400),
    _c("e16_farm_success", "POST", "/farm",
       body={"farm_name": "Contract Farm", "total_area": 2.5,
             "area_unit": "acres", "land_ownership": "own_land",
             "latitude": 20.0116, "longitude": 73.7908,
             "boundary_geom": NASHIK}, auth="valid", expect=201,
       phase="setup", note="farm_id feeds the crop/irrigation/soil cases"),

    # ── E17-E20 crop / irrigation / soil / consent ───────────────────────────
    _c("e17_crop_invalid", "POST", "/crop",
       body={"crop_name": "Grapes", "season": "monsoon"}, auth="valid",
       expect=400),
    _c("e17_crop_bad_owner", "POST", "/crop",
       body={"farm_id": "00000000-0000-0000-0000-000000000000",
             "crop_name": "Grapes", "sowing_date": "2025-06-15",
             "season": "kharif"}, auth="valid", expect=403,
       note="§7.10 ownership failure message"),
    _c("e17_crop_success", "POST", "/crop",
       body={"__farm_id__": True, "crop_name": "Grapes",
             "sowing_date": "2025-06-15", "season": "kharif"},
       auth="valid", expect=201),
    _c("e18_irrigation_invalid", "POST", "/irrigation",
       body={"__farm_id__": True, "irrigation_type": "magic"}, auth="valid",
       expect=400),
    _c("e18_irrigation_success", "POST", "/irrigation",
       body={"__farm_id__": True, "irrigation_type": "drip_irrigation",
             "water_source": "borewell"}, auth="valid", expect=201),
    _c("e19_soil_invalid", "POST", "/soil",
       body={"__farm_id__": True, "soil_type": "purple"}, auth="valid",
       expect=400),
    _c("e19_soil_success", "POST", "/soil",
       body={"__farm_id__": True, "soil_type": "black"}, auth="valid",
       expect=201),
    _c("e20_consent_invalid", "POST", "/consent", body={}, auth="valid",
       expect=400),
    _c("e20_consent_success", "POST", "/consent",
       body={"satellite_monitoring": True}, auth="valid", expect=200),

    # ── E21 dashboard ────────────────────────────────────────────────────────
    _c("e21_dashboard_success", "GET", "/dashboard", auth="valid", expect=200,
       note="§10.10 type-coercion minefield: numeric, date, raw json"),

    # ── E22-E24 chatbot ──────────────────────────────────────────────────────
    _c("e22_chat_empty_message", "POST", "/chatbot/chat",
       body={"message": "   "}, expect=400),
    _c("e23_reset_missing_session", "POST", "/chatbot/reset", body={},
       expect=400),
    _c("e23_reset_success", "POST", "/chatbot/reset",
       body={"session_id": "contract-test-session"}, expect=200),
    _c("e24_chatbot_health", "GET", "/chatbot/health", expect=200),
]


def case_count() -> int:
    return len(CASES)


if __name__ == "__main__":
    print(f"{case_count()} contract cases")
    for c in CASES:
        print(f"  {c['expect']:>3}  {c['method']:<4} {c['path']:<32} {c['id']}")

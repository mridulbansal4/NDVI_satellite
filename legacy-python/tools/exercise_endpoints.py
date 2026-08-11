"""exercise_endpoints.py — exercise EVERY endpoint and record what it returns.

Walks all 24 endpoints of PRD §10.1 against the Go backend, covering for each:
happy path, missing/invalid input, auth/unauthorized, and the edge cases the
PRD implies. Records the ACTUAL status code and response body for each case and
writes a Markdown table to docs/ENDPOINT_VERIFICATION.md.

Deliberately NOT exercised, with the reason recorded in the output:
  * POST /api/auth/send-otp happy path — hits a paid SMS gateway.
  * POST /chatbot/chat happy path — needs a running Ollama server.

Usage (Go server on :5001):
    venv/Scripts/python.exe tools/exercise_endpoints.py
"""

import datetime
import json
import pathlib
import sys
import uuid

import requests

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[1]))

import config  # noqa: E402,F401  (loads .env)
import os  # noqa: E402
import jwt as pyjwt  # noqa: E402

REPO = pathlib.Path(__file__).resolve().parents[2]
POLY = REPO / "testdata" / "polygons"
OUT = REPO / "docs" / "ENDPOINT_VERIFICATION.md"
BASE = os.getenv("GO_BASE", "http://127.0.0.1:5001")
SECRET = os.getenv("JWT_SECRET_KEY", "dev-secret-change-me")


def poly(name):
    return json.loads((POLY / f"{name}.json").read_text())


def mint(expired=False, secret=None):
    now = datetime.datetime.now(datetime.timezone.utc)
    iat = int(now.timestamp())
    return pyjwt.encode(
        {
            "sub": "00000000-0000-0000-0000-0000000000c1",
            "type": "access", "fresh": False,
            "iat": iat, "nbf": iat,
            "exp": iat - 60 if expired else iat + 604800,
            "jti": str(uuid.uuid4()),
        },
        secret or SECRET, algorithm="HS256",
    )


def hdr(mode):
    if mode is None:
        return {}
    if mode == "valid":
        return {"Authorization": "Bearer " + mint()}
    if mode == "expired":
        return {"Authorization": "Bearer " + mint(expired=True)}
    if mode == "badsig":
        return {"Authorization": "Bearer " + mint(secret="wrong-secret")}
    if mode == "malformed":
        return {"Authorization": "not-a-bearer-token"}
    raise ValueError(mode)


# (endpoint, case, method, path, body, query, auth, expected)
CASES = [
    ("E1  /health", "happy", "GET", "/health", None, None, None, 200),

    ("E7  /api/sample", "no analysis yet (cold cache)", "GET", "/api/sample", None,
     {"lat": "20.0116", "lng": "73.7908"}, None, 404),

    ("E2  /api/analyze", "happy", "POST", "/api/analyze",
     {"geometry": poly("nashik_vineyard")}, None, None, 200),
    ("E2  /api/analyze", "missing geometry", "POST", "/api/analyze", {}, None, None, 400),
    ("E2  /api/analyze", "geometry not an object", "POST", "/api/analyze",
     {"geometry": "nope"}, None, None, 400),
    ("E2  /api/analyze", "MultiPolygon rejected", "POST", "/api/analyze",
     {"geometry": {"type": "MultiPolygon", "coordinates": []}}, None, None, 400),
    ("E2  /api/analyze", "empty coordinates", "POST", "/api/analyze",
     {"geometry": {"type": "Polygon", "coordinates": []}}, None, None, 400),
    ("E2  /api/analyze", "ring with < 4 points", "POST", "/api/analyze",
     {"geometry": {"type": "Polygon", "coordinates": [[[0, 0], [1, 0], [0, 0]]]}},
     None, None, 400),
    ("E2  /api/analyze", "malformed coordinate pair", "POST", "/api/analyze",
     {"geometry": {"type": "Polygon", "coordinates": [[[0, 0], [1], [1, 1], [0, 0]]]}},
     None, None, 400),
    ("E2  /api/analyze", "longitude out of range", "POST", "/api/analyze",
     {"geometry": {"type": "Polygon",
                   "coordinates": [[[181.0, 0], [1, 0], [1, 1], [181.0, 0]]]}},
     None, None, 400),
    ("E2  /api/analyze", "latitude out of range", "POST", "/api/analyze",
     {"geometry": {"type": "Polygon",
                   "coordinates": [[[0, 91.5], [1, 0], [1, 1], [0, 91.5]]]}},
     None, None, 400),
    ("E2  /api/analyze", "no imagery (200 + error body)", "POST", "/api/analyze",
     {"geometry": poly("no_imagery")}, None, None, 200),
    ("E2  /api/analyze", "edge: water body (negative NDVI)", "POST", "/api/analyze",
     {"geometry": poly("water_body")}, None, None, 200),
    ("E2  /api/analyze", "edge: large field (grid auto-coarsening)", "POST", "/api/analyze",
     {"geometry": poly("large_field")}, None, None, 200),
    ("E2  /api/analyze", "edge: tiny plot", "POST", "/api/analyze",
     {"geometry": poly("tiny_plot")}, None, None, 200),

    ("E7  /api/sample", "happy (warm cache)", "GET", "/api/sample", None,
     {"lat": "20.0116", "lng": "73.7908"}, None, 200),
    ("E7  /api/sample", "band lower-cased", "GET", "/api/sample", None,
     {"lat": "20.0116", "lng": "73.7908", "band": "cvi"}, None, 200),
    ("E7  /api/sample", "non-numeric coords", "GET", "/api/sample", None,
     {"lat": "abc"}, None, 400),
    ("E7  /api/sample", "invalid band", "GET", "/api/sample", None,
     {"lat": "20.0116", "lng": "73.7908", "band": "NOPE"}, None, 400),

    ("E3  /api/analyze-dates", "happy", "POST", "/api/analyze-dates",
     {"geometry": poly("nashik_vineyard")}, None, None, 200),
    ("E3  /api/analyze-dates", "missing geometry", "POST", "/api/analyze-dates",
     {}, None, None, 400),
    ("E3  /api/analyze-dates", "invalid polygon", "POST", "/api/analyze-dates",
     {"geometry": {"type": "Point", "coordinates": [0, 0]}}, None, None, 400),

    ("E4  /api/analyze-day", "missing date", "POST", "/api/analyze-day",
     {"geometry": poly("nashik_vineyard")}, None, None, 400),
    ("E4  /api/analyze-day", "missing geometry", "POST", "/api/analyze-day",
     {"date": "2025-02-14"}, None, None, 400),
    ("E4  /api/analyze-day", "invalid polygon", "POST", "/api/analyze-day",
     {"geometry": {"type": "Point", "coordinates": [0, 0]}, "date": "2025-02-14"},
     None, None, 400),
    ("E4  /api/analyze-day", "no imagery for date (200 + error)", "POST", "/api/analyze-day",
     {"geometry": poly("nashik_vineyard"), "date": "1999-01-01"}, None, None, 200),

    ("E5  /api/analyze-radar-dates", "happy", "POST", "/api/analyze-radar-dates",
     {"geometry": poly("nashik_vineyard")}, None, None, 200),
    ("E5  /api/analyze-radar-dates", "missing geometry", "POST",
     "/api/analyze-radar-dates", {}, None, None, 400),
    ("E5  /api/analyze-radar-dates", "invalid polygon", "POST",
     "/api/analyze-radar-dates",
     {"geometry": {"type": "Point", "coordinates": [0, 0]}}, None, None, 400),

    ("E6  /api/analyze-radar", "happy (no date)", "POST", "/api/analyze-radar",
     {"geometry": poly("nashik_vineyard")}, None, None, 200),
    ("E6  /api/analyze-radar", "missing geometry", "POST", "/api/analyze-radar",
     {}, None, None, 400),
    ("E6  /api/analyze-radar", "no imagery, no date", "POST", "/api/analyze-radar",
     {"geometry": poly("no_imagery")}, None, None, 200),
    ("E6  /api/analyze-radar", "no imagery, with date", "POST", "/api/analyze-radar",
     {"geometry": poly("no_imagery"), "date": "2025-02-14"}, None, None, 200),

    ("E8  /api/auth/verify-token", "missing idToken", "POST",
     "/api/auth/verify-token", {}, None, None, 400),
    ("E8  /api/auth/verify-token", "invalid token", "POST",
     "/api/auth/verify-token", {"idToken": "not-a-real-token"}, None, None, 401),

    ("E9  /api/auth/send-otp", "too short", "POST", "/api/auth/send-otp",
     {"phone": "12345"}, None, None, 400),
    ("E9  /api/auth/send-otp", "non-digit", "POST", "/api/auth/send-otp",
     {"phone": "98765abcde"}, None, None, 400),

    ("E10 /api/auth/verify-otp", "missing fields", "POST", "/api/auth/verify-otp",
     {}, None, None, 400),
    ("E10 /api/auth/verify-otp", "wrong otp", "POST", "/api/auth/verify-otp",
     {"phone": "9000000001", "otp": "000000"}, None, None, 401),

    ("E12 /auth/login", "unknown mobile (401)", "POST", "/auth/login",
     {"mobile_number": "9000000099", "password": "whatever"}, None, None, 401),
    ("E11 /auth/signup", "bad mobile (422)", "POST", "/auth/signup",
     {"mobile_number": "123", "password": "secret1"}, None, None, 422),
    ("E11 /auth/signup", "short password (422)", "POST", "/auth/signup",
     {"mobile_number": "9000000001", "password": "abc"}, None, None, 422),
    ("E12 /auth/login", "bad mobile (422)", "POST", "/auth/login",
     {"mobile_number": "abc", "password": "x"}, None, None, 422),
    ("E12 /auth/login", "empty password (422)", "POST", "/auth/login",
     {"mobile_number": "9000000001", "password": ""}, None, None, 422),

    ("JWT middleware", "no Authorization header", "GET", "/dashboard", None, None,
     None, 401),
    ("JWT middleware", "non-Bearer header", "GET", "/dashboard", None, None,
     "malformed", 401),
    ("JWT middleware", "bad signature", "GET", "/dashboard", None, None,
     "badsig", 422),
    ("JWT middleware", "expired token", "GET", "/dashboard", None, None,
     "expired", 401),

    ("E13 /farmer/basic-details", "invalid body", "POST", "/farmer/basic-details",
     {"name": "", "preferred_language": "klingon"}, None, "valid", 400),
    ("E13 /farmer/basic-details", "unauthorized", "POST", "/farmer/basic-details",
     {"name": "X", "preferred_language": "hindi"}, None, None, 401),
    ("E14 /farmer/location", "bad pin", "POST", "/farmer/location",
     {"pin_code": "12", "village_name": "Testville"}, None, "valid", 400),
    ("E15 /farmer/pincode", "happy", "GET", "/farmer/pincode/422001", None, None,
     None, 200),
    ("E15 /farmer/pincode", "unknown pin (404)", "GET", "/farmer/pincode/000000",
     None, None, None, 404),
    ("E15 /farmer/pincode", "malformed pin (404)", "GET", "/farmer/pincode/12",
     None, None, None, 404),
    ("E16 /farm", "invalid body", "POST", "/farm",
     {"farm_name": "", "total_area": 0, "area_unit": "bushels",
      "land_ownership": "nope", "latitude": 200, "longitude": 400},
     None, "valid", 400),
    ("E17 /crop", "missing required", "POST", "/crop",
     {"crop_name": "Grapes", "season": "monsoon"}, None, "valid", 400),
    ("E18 /irrigation", "bad enum", "POST", "/irrigation",
     {"farm_id": "00000000-0000-0000-0000-000000000001",
      "irrigation_type": "magic"}, None, "valid", 400),
    ("E19 /soil", "bad enum", "POST", "/soil",
     {"farm_id": "00000000-0000-0000-0000-000000000001", "soil_type": "purple"},
     None, "valid", 400),
    ("E20 /consent", "missing field", "POST", "/consent", {}, None, "valid", 400),
    ("E21 /dashboard", "authorized, unknown farmer (404)", "GET", "/dashboard",
     None, None, "valid", 404),

    ("E22 /chatbot/chat", "empty message", "POST", "/chatbot/chat",
     {"message": "   "}, None, None, 400),
    ("E23 /chatbot/reset", "missing session_id", "POST", "/chatbot/reset",
     {}, None, None, 400),
    ("E23 /chatbot/reset", "happy", "POST", "/chatbot/reset",
     {"session_id": "verify-session"}, None, None, 200),
    ("E24 /chatbot/health", "happy", "GET", "/chatbot/health", None, None, None, 200),
]

NOT_EXERCISED = [
    ("E9  /api/auth/send-otp", "happy path",
     "would send a real, billed SMS through the nationalbulksms gateway"),
    ("E22 /chatbot/chat", "happy path",
     "needs a running Ollama server with the model pulled"),
    ("E8  /api/auth/verify-token", "happy path",
     "needs a genuine Firebase ID token minted by the phone-auth client"),
    ("E11-E21", "full 9-step onboarding happy path",
     "covered by tools/verify_onboarding.py, which walks all nine steps on "
     "BOTH backends and diffs them (0 failing)"),
]


def summarise(body, limit=180):
    s = json.dumps(body, ensure_ascii=False)
    if len(s) <= limit:
        return s
    # Summarise big analysis payloads rather than dumping thousands of cells.
    if isinstance(body, dict) and "features" in body:
        fs = body.get("farm_summary") or body.get("radar_summary") or {}
        bits = [f"{len(body['features'])} features"]
        if "confidence" in fs:
            bits.append(f"confidence={fs['confidence']}")
        if "scene_count" in fs:
            bits.append(f"scene_count={fs['scene_count']}")
        keys = ",".join(sorted(body.keys()))
        return "{" + "; ".join(bits) + f"; keys={keys}" + "}"
    return s[:limit] + "…"


def main() -> int:
    rows = []
    failures = 0
    session = requests.Session()

    for ep, case, method, path, body, query, auth, expect in CASES:
        try:
            r = session.request(method, BASE + path, json=body, params=query,
                                headers=hdr(auth), timeout=600)
            status = r.status_code
            try:
                payload = r.json()
            except ValueError:
                payload = {"__text__": r.text[:200]}
        except Exception as exc:
            status, payload = None, {"__error__": f"{type(exc).__name__}: {exc}"}

        ok = status == expect
        if not ok:
            failures += 1
        rows.append((ep, case, method, path, expect, status, ok, summarise(payload)))
        print(f"{'PASS' if ok else 'FAIL'}  {ep:32} {case:38} {status} (want {expect})")

    lines = [
        "# Endpoint Verification",
        "",
        "Generated by `legacy-python/tools/exercise_endpoints.py` against the Go",
        "backend. Every row is a real HTTP request; the status and body columns",
        "record what actually came back.",
        "",
        f"- Base URL: `{BASE}`",
        f"- Cases: {len(rows)}  |  passing: {len(rows) - failures}  |  failing: {failures}",
        "",
        "| Endpoint | Case | Method | Expected | Actual | Result | Response (abridged) |",
        "|---|---|---|---:|---:|---|---|",
    ]
    for ep, case, method, path, expect, status, ok, body in rows:
        safe = body.replace("|", "\\|")
        if len(safe) > 200:
            safe = safe[:200] + "…"
        lines.append(
            f"| `{ep}` | {case} | {method} | {expect} | {status} | "
            f"{'PASS' if ok else 'FAIL'} | `{safe}` |"
        )

    lines += ["", "## Not exercised", "",
              "| Endpoint | Case | Reason |", "|---|---|---|"]
    for ep, case, why in NOT_EXERCISED:
        lines.append(f"| `{ep}` | {case} | {why} |")
    lines.append("")

    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text("\n".join(lines), encoding="utf-8")
    print(f"\n{len(rows)} cases, {failures} failing -> {OUT}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

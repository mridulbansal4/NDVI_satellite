"""verify_onboarding.py — walk the full 9-step onboarding flow on BOTH backends.

PRD reference: PRAGYA_GO_MIGRATION_PRD.md §14 Phase 5 acceptance —
"full onboarding 9-step flow completes end-to-end".

Runs the identical sequence against Python (:5000) and Go (:5001) with two
separate test farmers, then diffs the responses structurally. Volatile values
(ids, tokens, timestamps) are masked; everything else must match exactly.

Also checks the cross-runtime guarantees of §8.2 / §8.3: a farmer created by one
backend logs into the other, and each backend accepts the other's JWT.

Side effects: creates and then deletes two test farmers. Teardown runs even on
failure.

Usage (both servers up):
    venv/Scripts/python.exe tools/verify_onboarding.py
"""

import json
import pathlib
import sys

import requests

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[1]))

import config  # noqa: E402,F401  (loads .env)
from db.pool import DBConnection, init_pool  # noqa: E402

REPO = pathlib.Path(__file__).resolve().parents[2]
POLY = REPO / "testdata" / "polygons"
PY = "http://127.0.0.1:5000"
GO = "http://127.0.0.1:5001"

PY_MOBILE = "9000000090"
GO_MOBILE = "9000000091"
PASSWORD = "onboarding-test-pw"

# Steps where Go is EXPECTED to differ from Python, with the reason.
#
# K11 (owner-approved): api.postalpincode.in TCP-resets the default
# python-requests User-Agent, so the Python backend silently stores empty
# state/district/taluka. Go sets an explicit UA and stores the real values.
# This is the one sanctioned deviation from PRD §0.8.
EXPECTED_DIVERGENCE = {
    "3 location": "K11 — Python's PIN lookup is UA-blocked and stores empty "
                  "state/district/taluka; Go resolves them (owner-approved)",
}

# Values that legitimately differ between two runs.
VOLATILE = {
    # the two runs use different test farmers by design
    "mobile_number",
    "id", "farmer_id", "farm_id", "token", "consent_id",
    "created_at", "consented_at",
}


def cleanup():
    try:
        init_pool()
    except Exception:
        pass
    try:
        with DBConnection() as conn:
            with conn.cursor() as cur:
                cur.execute("DELETE FROM farmers WHERE mobile_number = ANY(%s)",
                            ([PY_MOBILE, GO_MOBILE],))
                print(f"[teardown] removed {cur.rowcount} test farmer row(s)")
    except Exception as exc:
        print(f"[teardown] WARNING: {exc}", file=sys.stderr)


def mask(v):
    """Recursively blank volatile values so two runs are comparable."""
    if isinstance(v, dict):
        return {k: ("<masked>" if k in VOLATILE else mask(x)) for k, x in v.items()}
    if isinstance(v, list):
        return [mask(x) for x in v]
    return v


def walk(base, mobile):
    """Run the nine steps and return [(label, status, body), ...]."""
    out = []
    s = requests.Session()

    def rec(label, resp):
        try:
            body = resp.json()
        except ValueError:
            body = {"__text__": resp.text[:200]}
        out.append((label, resp.status_code, body))
        return body

    body = rec("1 signup", s.post(f"{base}/auth/signup",
               json={"mobile_number": mobile, "password": PASSWORD}, timeout=60))
    token = body.get("token")
    auth = {"Authorization": f"Bearer {token}"}

    rec("1b duplicate signup (409)", s.post(f"{base}/auth/signup",
        json={"mobile_number": mobile, "password": PASSWORD}, timeout=60))
    rec("1c login", s.post(f"{base}/auth/login",
        json={"mobile_number": mobile, "password": PASSWORD}, timeout=60))
    rec("1d wrong password (401)", s.post(f"{base}/auth/login",
        json={"mobile_number": mobile, "password": "nope"}, timeout=60))
    rec("1e unknown mobile (401)", s.post(f"{base}/auth/login",
        json={"mobile_number": "9000000099", "password": PASSWORD}, timeout=60))

    rec("2 basic-details", s.post(f"{base}/farmer/basic-details",
        json={"name": "Onboarding Test", "age": 41, "gender": "male",
              "preferred_language": "marathi"}, headers=auth, timeout=60))
    rec("3 location", s.post(f"{base}/farmer/location",
        json={"pin_code": "422001", "village_name": "Testville"},
        headers=auth, timeout=60))

    farm = rec("4+5 farm", s.post(f"{base}/farm",
        json={"farm_name": "Verification Farm", "total_area": 2.5,
              "area_unit": "acres", "land_ownership": "own_land",
              "latitude": 20.0116, "longitude": 73.7908,
              "boundary_geom": json.loads((POLY / "nashik_vineyard.json").read_text())},
        headers=auth, timeout=60))
    farm_id = farm.get("id")

    rec("6 crop", s.post(f"{base}/crop",
        json={"farm_id": farm_id, "crop_name": "Grapes",
              "sowing_date": "2025-06-15", "season": "kharif"},
        headers=auth, timeout=60))
    rec("6b crop wrong owner (403)", s.post(f"{base}/crop",
        json={"farm_id": "00000000-0000-0000-0000-000000000000",
              "crop_name": "Grapes", "sowing_date": "2025-06-15",
              "season": "kharif"}, headers=auth, timeout=60))
    rec("7 irrigation", s.post(f"{base}/irrigation",
        json={"farm_id": farm_id, "irrigation_type": "drip_irrigation",
              "water_source": "borewell"}, headers=auth, timeout=60))
    rec("8 soil", s.post(f"{base}/soil",
        json={"farm_id": farm_id, "soil_type": "black"}, headers=auth, timeout=60))
    rec("8b soil upsert (idempotent)", s.post(f"{base}/soil",
        json={"farm_id": farm_id, "soil_type": "red"}, headers=auth, timeout=60))
    rec("9 consent", s.post(f"{base}/consent",
        json={"satellite_monitoring": True}, headers=auth, timeout=60))
    rec("10 dashboard", s.get(f"{base}/dashboard", headers=auth, timeout=60))

    return out, token


def main() -> int:
    cleanup()
    failures = 0
    try:
        print("=== 9-step onboarding: Python vs Go ===\n")
        py_steps, py_token = walk(PY, PY_MOBILE)
        go_steps, go_token = walk(GO, GO_MOBILE)

        for (plabel, pstat, pbody), (glabel, gstat, gbody) in zip(py_steps, go_steps):
            pm, gm = mask(pbody), mask(gbody)
            ok = pstat == gstat and pm == gm
            if plabel in EXPECTED_DIVERGENCE:
                # Status must still match; only the body is allowed to differ.
                status_ok = pstat == gstat
                if not status_ok:
                    failures += 1
                print(f"{'DIVERGE' if status_ok else 'FAIL'}  {plabel:32} "
                      f"py={pstat} go={gstat}")
                print(f"        expected: {EXPECTED_DIVERGENCE[plabel]}")
                print(f"        py: {json.dumps(pm)[:180]}")
                print(f"        go: {json.dumps(gm)[:180]}")
                continue
            if not ok:
                failures += 1
            print(f"{'PASS' if ok else 'FAIL'}  {plabel:32} py={pstat} go={gstat}")
            if not ok:
                print(f"        py: {json.dumps(pm)[:220]}")
                print(f"        go: {json.dumps(gm)[:220]}")

        # Cross-runtime checks (§8.2, §8.3).
        print("\n=== cross-runtime credential + token interchange ===")
        checks = [
            ("go login with python-created farmer",
             requests.post(f"{GO}/auth/login",
                           json={"mobile_number": PY_MOBILE, "password": PASSWORD},
                           timeout=60), 200),
            ("python login with go-created farmer",
             requests.post(f"{PY}/auth/login",
                           json={"mobile_number": GO_MOBILE, "password": PASSWORD},
                           timeout=60), 200),
            ("go /dashboard with python-issued JWT",
             requests.get(f"{GO}/dashboard",
                          headers={"Authorization": f"Bearer {py_token}"}, timeout=60), 200),
            ("python /dashboard with go-issued JWT",
             requests.get(f"{PY}/dashboard",
                          headers={"Authorization": f"Bearer {go_token}"}, timeout=60), 200),
        ]
        for label, resp, want in checks:
            ok = resp.status_code == want
            if not ok:
                failures += 1
            print(f"{'PASS' if ok else 'FAIL'}  {label:40} {resp.status_code} (want {want})")
    finally:
        cleanup()

    print(f"\n{failures} failing")
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())

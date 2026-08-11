"""dump_contract.py — capture Layer-2 HTTP contract goldens from Flask.

PRD reference: PRAGYA_GO_MIGRATION_PRD.md §12.2, §14 Phase 0.

Fires the frozen corpus (contract_corpus.py) at the running Python backend and
records every status code + response body to testdata/golden/. The Go contract
runner replays the same corpus against the Go backend and diffs against these
files. This MUST be run before the Python backend is moved or modified.

Usage (from the backend directory, with the Flask app running on :5000):

    venv/Scripts/python.exe app.py                    # terminal 1
    venv/Scripts/python.exe tools/dump_contract.py    # terminal 2

Side effects: creates and then deletes a contract-test farmer (and its farm,
crop, irrigation, soil and consent rows) so that the 409-duplicate and
403-ownership branches can be captured. Teardown runs even on failure.
"""

import argparse
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

from tools.contract_corpus import CASES, TEST_MOBILE  # noqa: E402

REPO = pathlib.Path(__file__).resolve().parents[2]
OUT = REPO / "testdata" / "golden"

JWT_SECRET = os.getenv("JWT_SECRET_KEY", "dev-secret-change-me")

# Values discovered during the setup phase and substituted into later cases.
STATE: dict = {"token": None, "farmer_id": None, "farm_id": None}


def _mint(sub: str, *, expired: bool = False, secret: str = JWT_SECRET) -> str:
    """Mint a Flask-JWT-Extended-shaped token (§8.3)."""
    now = datetime.datetime.now(datetime.timezone.utc)
    iat = int(now.timestamp())
    exp = iat - 60 if expired else iat + 604800
    return pyjwt.encode(
        {
            "sub": sub, "type": "access", "fresh": False,
            "iat": iat, "nbf": iat, "exp": exp, "jti": str(uuid.uuid4()),
        },
        secret,
        algorithm="HS256",
    )


def _headers(auth) -> dict:
    if auth is None:
        return {}
    if auth == "valid":
        return {"Authorization": f"Bearer {STATE['token']}"}
    if auth == "malformed":
        return {"Authorization": STATE["token"] or "xyz"}  # no "Bearer " prefix
    if auth == "badsig":
        return {"Authorization": f"Bearer {_mint('00000000-0000-0000-0000-000000000000', secret='wrong-secret')}"}
    if auth == "expired":
        sub = STATE["farmer_id"] or "00000000-0000-0000-0000-000000000000"
        return {"Authorization": f"Bearer {_mint(sub, expired=True)}"}
    raise ValueError(f"unknown auth mode {auth!r}")


def _resolve_body(body):
    """Substitute the runtime farm_id placeholder."""
    if not isinstance(body, dict):
        return body
    out = dict(body)
    if out.pop("__farm_id__", None):
        out["farm_id"] = STATE["farm_id"]
    return out


def _teardown() -> None:
    """Remove the contract-test farmer and everything hanging off it."""
    try:
        from db.pool import init_pool, DBConnection
        try:
            init_pool()
        except Exception:
            pass  # already initialised
        with DBConnection() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "DELETE FROM farmers WHERE mobile_number = %s", (TEST_MOBILE,)
                )
                print(f"[teardown] removed {cur.rowcount} contract-test farmer row(s)")
    except Exception as exc:
        print(f"[teardown] WARNING could not clean up: {exc}", file=sys.stderr)


def run(base: str) -> int:
    OUT.mkdir(parents=True, exist_ok=True)
    _teardown()  # start from a clean slate

    session = requests.Session()
    summary = []
    failures = 0

    try:
        for case in CASES:
            url = base.rstrip("/") + case["path"]
            body = _resolve_body(case["body"])
            try:
                resp = session.request(
                    case["method"], url, json=body if case["method"] == "POST" else None,
                    params=case["query"], headers=_headers(case["auth"]), timeout=300,
                )
            except Exception as exc:
                print(f"  !! {case['id']}: request failed: {exc}")
                failures += 1
                continue

            try:
                payload = resp.json()
            except ValueError:
                payload = {"__non_json_body__": resp.text}

            golden = {
                "id": case["id"], "method": case["method"], "path": case["path"],
                "query": case["query"], "body": body, "auth": case["auth"],
                "status": resp.status_code, "response": payload,
                "note": case["note"],
            }
            (OUT / f"{case['id']}.json").write_text(
                json.dumps(golden, indent=2, ensure_ascii=False, sort_keys=True),
                encoding="utf-8",
            )

            ok = resp.status_code == case["expect"]
            if not ok:
                failures += 1
            summary.append((case["id"], case["expect"], resp.status_code, ok))
            print(f"  {'ok ' if ok else 'MISMATCH'} {case['id']}: "
                  f"expected {case['expect']}, got {resp.status_code}")

            # Harvest state produced by the setup cases.
            if case["id"] == "e11_signup_success" and isinstance(payload, dict):
                STATE["token"] = payload.get("token")
                STATE["farmer_id"] = payload.get("farmer_id")
            if case["id"] == "e12_login_success" and isinstance(payload, dict):
                STATE["token"] = payload.get("token") or STATE["token"]
                STATE["farmer_id"] = payload.get("farmer_id") or STATE["farmer_id"]
            if case["id"] == "e16_farm_success" and isinstance(payload, dict):
                STATE["farm_id"] = payload.get("id")
    finally:
        _teardown()

    manifest = {
        "captured_from": base,
        "case_count": len(CASES),
        "status_mismatches": failures,
        "cases": [
            {"id": i, "expected": e, "actual": a, "match": m}
            for i, e, a, m in summary
        ],
    }
    (OUT / "manifest.json").write_text(
        json.dumps(manifest, indent=2), encoding="utf-8"
    )

    print(f"\n{len(summary)} cases captured to {OUT}")
    print(f"{failures} status mismatch(es) against the documented expectation")
    return 0


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="http://127.0.0.1:5000")
    args = ap.parse_args()
    return run(args.base)


if __name__ == "__main__":
    raise SystemExit(main())

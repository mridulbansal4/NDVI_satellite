"""verify_endpoints.py — side-by-side endpoint verification, Go vs Python.

Fires each endpoint at BOTH backends on the same day with the same inputs and
reports status codes plus a structural/numeric diff. Unlike the golden-based
contract runner this is immune to the 90-day lookback window drifting, because
both backends see the same "today".

Usage (both servers up):
    venv/Scripts/python.exe tools/verify_endpoints.py
"""

import json
import pathlib
import sys

import requests

REPO = pathlib.Path(__file__).resolve().parents[2]
POLY = REPO / "testdata" / "polygons"
GO = "http://127.0.0.1:5001"
PY = "http://127.0.0.1:5000"
TOL = 1e-6


def poly(name):
    return json.loads((POLY / f"{name}.json").read_text())


def fire(base, method, path, body=None, params=None, timeout=600):
    try:
        r = requests.request(method, base + path, json=body, params=params, timeout=timeout)
        try:
            return r.status_code, r.json()
        except ValueError:
            return r.status_code, {"__text__": r.text[:200]}
    except Exception as exc:
        return None, {"__error__": f"{type(exc).__name__}: {exc}"}


def numeric_diff(a, b, path=""):
    """Yield (path, go, py) for every numeric mismatch beyond TOL."""
    out = []
    if isinstance(b, dict) and isinstance(a, dict):
        for k in set(a) | set(b):
            if k in ("ndvi_tile_url", "tile_url", "index_tiles", "radar_tiles") or k.endswith("_tile_url"):
                continue
            if k not in a:
                out.append((path + "." + k, "<missing>", "present"))
            elif k not in b:
                out.append((path + "." + k, "present", "<missing>"))
            else:
                out += numeric_diff(a[k], b[k], path + "." + k)
    elif isinstance(a, list) and isinstance(b, list):
        if len(a) != len(b):
            out.append((path, f"len={len(a)}", f"len={len(b)}"))
        else:
            for i, (x, y) in enumerate(zip(a, b)):
                out += numeric_diff(x, y, f"{path}[{i}]")
    elif isinstance(a, (int, float)) and isinstance(b, (int, float)) \
            and not isinstance(a, bool) and not isinstance(b, bool):
        if abs(a - b) > TOL:
            out.append((path, a, b))
    elif a != b:
        out.append((path, a, b))
    return out


CASES = [
    ("E1  GET  /health", "GET", "/health", None, None),
    ("E3  POST /api/analyze-dates", "POST", "/api/analyze-dates",
     {"geometry": poly("nashik_vineyard")}, None),
    ("E5  POST /api/analyze-radar-dates", "POST", "/api/analyze-radar-dates",
     {"geometry": poly("nashik_vineyard")}, None),
    ("E6  POST /api/analyze-radar", "POST", "/api/analyze-radar",
     {"geometry": poly("nashik_vineyard")}, None),
    ("E2  POST /api/analyze (water_body)", "POST", "/api/analyze",
     {"geometry": poly("water_body")}, None),
    ("E2  POST /api/analyze (no imagery)", "POST", "/api/analyze",
     {"geometry": poly("no_imagery")}, None),
    ("E7  GET  /api/sample", "GET", "/api/sample", None,
     {"lat": "20.0116", "lng": "73.7908"}),
    ("E7  GET  /api/sample (band=cvi)", "GET", "/api/sample", None,
     {"lat": "20.0116", "lng": "73.7908", "band": "cvi"}),
    ("E7  GET  /api/sample (bad band)", "GET", "/api/sample", None,
     {"lat": "20.0116", "lng": "73.7908", "band": "NOPE"}),
    ("E24 GET  /chatbot/health", "GET", "/chatbot/health", None, None),
]


def main() -> int:
    failures = 0
    for label, method, path, body, params in CASES:
        gs, gb = fire(GO, method, path, body, params)
        ps, pb = fire(PY, method, path, body, params)

        status_ok = gs == ps
        # /health capability flags legitimately differ between processes.
        if path == "/health":
            for d in (gb, pb):
                d.pop("gee_ready", None)
                d.pop("firebase_ready", None)

        diffs = numeric_diff(gb, pb)
        ok = status_ok and not diffs
        if not ok:
            failures += 1

        print(f"{'PASS' if ok else 'FAIL'}  {label}")
        print(f"        status go={gs} py={ps}")
        if not status_ok:
            print(f"        go body: {json.dumps(gb)[:200]}")
            print(f"        py body: {json.dumps(pb)[:200]}")
        for p, a, b in diffs[:6]:
            print(f"        diff {p}: go={a!r} py={b!r}")
        if len(diffs) > 6:
            print(f"        … and {len(diffs) - 6} more")

    print(f"\n{len(CASES)} cases, {failures} failing")
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())

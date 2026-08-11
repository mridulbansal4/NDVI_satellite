"""dump_numeric.py — Layer-3 numeric parity goldens.

PRD reference: PRAGYA_GO_MIGRATION_PRD.md §12.3.

Captures the full /api/analyze and /api/analyze-radar responses for every
fixture polygon, so the Go pipeline can be asserted to within 1e-6 on per-cell
values and farm-summary means (1e-9 for smoothed values).

Run with the Flask backend up:
    venv/Scripts/python.exe tools/dump_numeric.py
"""

import argparse
import json
import pathlib
import sys

import requests

REPO = pathlib.Path(__file__).resolve().parents[2]
POLY = REPO / "testdata" / "polygons"
OUT = REPO / "testdata" / "golden" / "numeric"

FIXTURES = [
    "nashik_vineyard",
    "tiny_plot",
    "large_field",
    "water_body",
    "cloudy_region",
]


def capture(base: str, name: str) -> dict:
    geometry = json.loads((POLY / f"{name}.json").read_text())
    out = {"polygon": name, "geometry": geometry}
    for label, path, body in [
        ("analyze", "/api/analyze", {"geometry": geometry}),
        ("radar", "/api/analyze-radar", {"geometry": geometry}),
    ]:
        try:
            r = requests.post(base.rstrip("/") + path, json=body, timeout=600)
            out[label] = {"status": r.status_code, "response": r.json()}
            feats = out[label]["response"].get("features")
            n = len(feats) if isinstance(feats, list) else 0
            print(f"  {name}/{label}: {r.status_code}, {n} cells")
        except Exception as exc:
            out[label] = {"status": None, "error": f"{type(exc).__name__}: {exc}"}
            print(f"  {name}/{label}: FAILED {type(exc).__name__}: {exc}")
    return out


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="http://127.0.0.1:5000")
    ap.add_argument("--only", nargs="*", default=None)
    args = ap.parse_args()

    OUT.mkdir(parents=True, exist_ok=True)
    targets = args.only or FIXTURES
    for name in targets:
        print(f"capturing {name}…")
        payload = capture(args.base, name)
        (OUT / f"{name}.json").write_text(
            json.dumps(payload, indent=2, ensure_ascii=False, sort_keys=True),
            encoding="utf-8",
        )
    print(f"\nwrote {len(targets)} numeric goldens to {OUT}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

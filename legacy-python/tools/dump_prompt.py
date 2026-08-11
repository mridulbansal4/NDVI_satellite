"""dump_prompt.py — Krishi Mitra system-prompt render fixture.

PRD reference: PRAGYA_GO_MIGRATION_PRD.md §10.11.

prompts/system_prompt.py must be ported to a Go text/template
character-for-character, including the non-ASCII typography. This dumps the
rendered output for two fixed inputs (a populated field and the routes.py
fallback constants) so the Go test can diff its own render against it.

Run from the backend directory:
    venv/Scripts/python.exe tools/dump_prompt.py
"""

import json
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parents[1]))

from chatbot.prompts import build_system_prompt  # noqa: E402
from chatbot.routes import _FALLBACK_FARM, _FALLBACK_HEATMAP  # noqa: E402

REPO = pathlib.Path(__file__).resolve().parents[2]
OUT = REPO / "internal" / "chatbot" / "testdata"

POPULATED_FARM = {
    "fieldName": "Ramesh's Vineyard",
    "area": 2.5,
    "date": "2025-02-14",
    "confidence": 0.7314,
    "cleanScenes": 7,
    "cvi": 0.6259,
    "ndvi": 0.8651,
    "evi": 0.5013,
    "savi": 0.4733,
    "ndmi": 0.3288,
    "gndvi": 0.775,
}

POPULATED_HEATMAP = {
    "stressedPct": 12.5,
    "stressedLocation": "the north-west corner",
    "moderatePct": 30.0,
    "moderateLocation": "the central strip",
    "healthyPct": 57.5,
    "healthyLocation": "the southern half",
}


def main() -> int:
    OUT.mkdir(parents=True, exist_ok=True)
    cases = {
        "populated": {
            "farm_data": POPULATED_FARM,
            "heatmap_data": POPULATED_HEATMAP,
            "rendered": build_system_prompt(POPULATED_FARM, POPULATED_HEATMAP),
        },
        "fallback": {
            "farm_data": _FALLBACK_FARM,
            "heatmap_data": _FALLBACK_HEATMAP,
            "rendered": build_system_prompt(_FALLBACK_FARM, _FALLBACK_HEATMAP),
        },
    }
    (OUT / "system_prompt.json").write_text(
        json.dumps(cases, indent=2, ensure_ascii=False), encoding="utf-8"
    )
    for name, case in cases.items():
        print(f"{name}: {len(case['rendered'])} chars")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

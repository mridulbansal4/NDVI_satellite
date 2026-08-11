"""dump_crypto.py — Werkzeug password-hash golden fixtures.

PRD reference: PRAGYA_GO_MIGRATION_PRD.md §8.2.

Existing rows in farmers.password_hash cannot be re-hashed (the plaintext is
gone), so the Go port must verify Werkzeug's format directly. These fixtures are
the acceptance test for internal/crypto/werkzeug.go: if the dkLen or
salt-encoding assumptions are wrong, the table test fails immediately, which is
the entire point.

Run from the backend directory:
    venv/Scripts/python.exe tools/dump_crypto.py
"""

import json
import pathlib
import sys

from werkzeug.security import check_password_hash, generate_password_hash

REPO = pathlib.Path(__file__).resolve().parents[2]
OUT = REPO / "internal" / "crypto" / "testdata"

PASSWORDS = ["password123", "hunter2", "aA1!£€ \U0001f600", "x" * 72]
METHODS = ["scrypt", "pbkdf2:sha256"]


def main() -> int:
    OUT.mkdir(parents=True, exist_ok=True)
    cases = []
    for pw in PASSWORDS:
        for method in METHODS:
            h = generate_password_hash(pw, method=method)
            assert check_password_hash(h, pw), "python cannot verify its own hash"
            assert not check_password_hash(h, pw + "x"), "wrong password verified"
            cases.append({"password": pw, "method": method, "hash": h})

    # Also record what the DEFAULT method produces, since that is what rows
    # written by the running backend actually contain.
    default_hash = generate_password_hash("password123")
    cases.append(
        {"password": "password123", "method": "<default>", "hash": default_hash}
    )

    payload = {
        "werkzeug_version": __import__("importlib.metadata", fromlist=["version"]).version(
            "werkzeug"
        ),
        "default_method_prefix": default_hash.split("$")[0],
        "cases": cases,
    }
    (OUT / "werkzeug_hashes.json").write_text(
        json.dumps(payload, indent=2, ensure_ascii=False), encoding="utf-8"
    )
    print(f"wrote {len(cases)} cases to {OUT / 'werkzeug_hashes.json'}")
    print("werkzeug", payload["werkzeug_version"])
    print("default method prefix:", payload["default_method_prefix"])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

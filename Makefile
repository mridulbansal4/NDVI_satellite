# Makefile — Project Pragya Go backend
#
# PRD reference: PRAGYA_GO_MIGRATION_PRD.md §12.5, §14.
#
# `make verify` is the gate every migration phase must pass before the next one
# starts. It runs, in order: vet, staticcheck, race-enabled tests, contract suite.

GO          ?= go
BIN         ?= bin/server
# NOT ./... — frontend/node_modules vendors a Go package (flatted/golang) that
# would otherwise be built, vetted and tested. frontend/ must not be modified
# (PRD §0.2), so the package set is scoped here instead.
PKG         := ./cmd/... ./internal/... ./tools/...
PY          ?= legacy-python/venv/Scripts/python.exe
PY_BASE     ?= http://127.0.0.1:5000
GO_BASE     ?= http://127.0.0.1:5001

.PHONY: help build run test test-race vet staticcheck lint contract verify \
        fixtures fixtures-graphs fixtures-crypto fixtures-prompt \
        fixtures-contract fixtures-numeric docker clean tidy

help:
	@echo "build            build the server binary"
	@echo "run              run the server"
	@echo "test             go test ./..."
	@echo "test-race        go test ./... -race"
	@echo "vet              go vet ./..."
	@echo "staticcheck      staticcheck ./..."
	@echo "contract         diff Go responses against the captured Python goldens"
	@echo "verify           vet + staticcheck + test-race + contract  (the phase gate)"
	@echo "fixtures         regenerate every golden from the Python backend"
	@echo "docker           build the container image"

build:
	$(GO) build -trimpath -o $(BIN) ./cmd/server

run:
	$(GO) run ./cmd/server

test:
	$(GO) test $(PKG)

test-race:
	$(GO) test $(PKG) -race

vet:
	$(GO) vet $(PKG)

staticcheck:
	@command -v staticcheck >/dev/null 2>&1 || { \
		echo "staticcheck not installed; run: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
		exit 1; }
	staticcheck $(PKG)

lint: vet staticcheck

# Replays the frozen corpus against the Go server and diffs each response
# against testdata/golden/. Exits non-zero on any mismatch.
contract:
	$(GO) run ./tools/contract -go-base=$(GO_BASE)

# Same, but live-diffs Go against a running Python backend instead of the
# committed goldens. Useful while the Python reference is still around.
contract-live:
	$(GO) run ./tools/contract -go-base=$(GO_BASE) -python-base=$(PY_BASE)

verify: vet staticcheck test-race contract

# ── Golden fixture regeneration (requires the Python backend + venv) ─────────
# These MUST be run against the reference implementation, never against Go.
fixtures: fixtures-graphs fixtures-crypto fixtures-prompt fixtures-contract fixtures-numeric

fixtures-graphs:
	cd legacy-python && $(PY) tools/dump_graphs.py

fixtures-crypto:
	cd legacy-python && $(PY) tools/dump_crypto.py

fixtures-prompt:
	cd legacy-python && $(PY) tools/dump_prompt.py

# Needs the Flask app listening on $(PY_BASE).
fixtures-contract:
	cd legacy-python && $(PY) tools/dump_contract.py --base=$(PY_BASE)

fixtures-numeric:
	cd legacy-python && $(PY) tools/dump_numeric.py --base=$(PY_BASE)

docker:
	docker build -t pragya-backend:dev .

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin

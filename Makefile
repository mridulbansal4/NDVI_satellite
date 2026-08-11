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

verify: vet staticcheck test-race contract

# ── Golden fixture regeneration ─────────────────────────────────────────────
#
# The Python reference implementation was deleted at the v2.0.0-go cutover, so
# the fixtures can no longer be regenerated in place. Every fixture under
# testdata/ and internal/*/testdata/ is committed and is what the Go tests
# assert against.
#
# To regenerate (e.g. after an Earth Engine API change), restore the reference
# implementation from history and re-run its capture tools:
#
#     git checkout v2.0.0-go~1 -- legacy-python
#     cd legacy-python && venv/Scripts/python.exe tools/dump_graphs.py
#     ...  dump_contract.py / dump_numeric.py / dump_crypto.py / gen_prompt.py
#     git rm -r --cached legacy-python && rm -rf legacy-python
fixtures:
	@echo "The Python reference was removed at v2.0.0-go."
	@echo "See the comment above this target for how to restore it and recapture."
	@exit 1

docker:
	docker build -t pragya-backend:dev .

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin

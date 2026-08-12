# Dockerfile — Project Pragya Go backend
# PRD reference: PRAGYA_GO_MIGRATION_PRD.md §13.10.
#
# Secrets (gee-service-account.json, serviceAccountKey.json, .env) are MOUNTED
# at runtime, never baked into the image.

# The tag must satisfy the `go` directive in go.mod (currently 1.25.0).
# A lower tag either fails outright under GOTOOLCHAIN=local or silently
# downloads a different toolchain at build time, which defeats the pin.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/server /server
EXPOSE 5000
# The server exposes /health; it returns 200 even when GEE and Firebase are
# unavailable, which is deliberate (§5.5) — the container is up as soon as HTTP
# is serving, and gee_ready/firebase_ready report degraded capability.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/server", "-healthcheck"]
ENTRYPOINT ["/server"]

# syntax=docker/dockerfile:1

# ── Build stage ───────────────────────────────────────────────────────────────
# Use the Debian-based image instead of Alpine: it ships a complete, up-to-date
# CA bundle which prevents x509 "certificate signed by unknown authority" errors
# when downloading modules from proxy.golang.org.
FROM golang:1.25 AS builder

# git may be required by modules that embed VCS metadata.
# ca-certificates and tzdata are already present on the Debian image; we install
# tzdata explicitly so the data directory is available to copy into distroless.
RUN apt-get update \
    && apt-get install -y --no-install-recommends git tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Cache dependency layer separately from source code.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# Produce a fully-static binary (no libc dependency) so it runs on distroless/static.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /lume ./cmd/lume

# ── Runtime stage ─────────────────────────────────────────────────────────────
# gcr.io/distroless/static-debian12:nonroot has no shell, no package manager,
# and runs as a non-root user (uid 65532) by default.
FROM gcr.io/distroless/static-debian12:nonroot

# Timezone data and root certificates are required for GCP API calls.
COPY --from=builder /usr/share/zoneinfo        /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /lume /lume

EXPOSE 3000

ENTRYPOINT ["/lume"]


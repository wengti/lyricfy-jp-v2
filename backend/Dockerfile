# Production image for the Go API (Cloud Run).
#
# Build (from backend/):  docker build -t lyricfy-api .
# Run locally:            docker run --rm -p 8080:8080 \
#                           -v "$(pwd)/config.local.yml:/config/config.yml:ro" \
#                           -e CONFIG_PATH=/config/config.yml lyricfy-api
#
# No config or secrets are baked in. On Cloud Run the config comes from Parameter Manager
# (CONFIG_PARAMETER); locally, from the file at CONFIG_PATH.

# ---- Build ----
FROM golang:1.26 AS build
WORKDIR /src

# Download modules in their own layer so code changes don't re-download them
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Static binary (no cgo) so it runs on distroless/static; -s -w strips debug info
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---- Runtime ----
# distroless/static: no shell or package manager, but includes CA certificates (for
# HTTPS to Neon, OpenRouter, Spotify) and tzdata. The :nonroot tag runs as UID 65532.
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/server /server

# Cloud Run sets PORT, which overrides server.port in the config
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]

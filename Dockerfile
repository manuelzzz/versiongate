# syntax=docker/dockerfile:1

# Build stage: compiles a static server binary. The Go toolchain and
# module cache never make it into the final image.
FROM golang:1.23-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/versiongate-server ./cmd/server

# Final stage: distroless base has no shell, no package manager, and
# already runs as a non-root user — a minimal attack surface for a
# self-hosted, network-facing service.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/versiongate-server /versiongate-server

EXPOSE 8888
ENTRYPOINT ["/versiongate-server"]

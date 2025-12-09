### STAGE 1: base image
ARG BASE_IMAGE_REGISTRY=cgr.dev
ARG BUILDPLATFORM
FROM --platform=$BUILDPLATFORM $BASE_IMAGE_REGISTRY/chainguard/go:latest AS builder
ARG TARGETARCH
ARG TARGETPLATFORM
ARG BUILDPLATFORM

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN --mount=type=cache,target=/root/go/pkg/mod,rw      \
    --mount=type=cache,target=/root/.cache/go-build,rw \
     go mod download

# Copy the go source
COPY cmd cmd
COPY pkg pkg
COPY internal internal
COPY api api

# Build
ARG LDFLAGS
RUN --mount=type=cache,target=/root/go/pkg/mod,rw             \
    --mount=type=cache,target=/root/.cache/go-build,rw        \
    echo "Building on $BUILDPLATFORM -> linux/$TARGETARCH" && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -ldflags "$LDFLAGS" -o databricks-mcp cmd/databricks-mcp/main.go

### STAGE 2: final image
FROM gcr.io/distroless/static:nonroot
ARG TARGETPLATFORM

WORKDIR /
COPY --from=builder /workspace/databricks-mcp /databricks-mcp
USER 65532:65532
ARG VERSION

LABEL org.opencontainers.image.source=https://github.com/kagent-dev/kagent
LABEL org.opencontainers.image.description="Databricks MCP server for Kagent DataSources"
LABEL org.opencontainers.image.authors="Kagent Creators"
LABEL org.opencontainers.image.version="$VERSION"

EXPOSE 8080
ENTRYPOINT ["/databricks-mcp"]

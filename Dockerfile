# Build the manager binary
# Use dhi.io/golang:1.26.8-alpine3.24
# Refer to https://hub.docker.com/hardened-images/catalog/dhi/golang/images/ for more details
FROM dhi.io/golang@sha256:9aaf4c5713faa338e6adaef0e67c23d4b2f033270f44b76aa2616625304d90c5 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/ internal/

# Build the manager for the target image platform
# BuildKit supplies TARGETOS and TARGETARCH, including for --platform builds
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

# Use dhi.io/static:20250419-debian13
# Refer to https://hub.docker.com/hardened-images/catalog/dhi/static/images/ for more details
FROM dhi.io/static@sha256:98ef7a853608577e8d66dad1d25ada75d745d782f28d84e9ecfb85dfeb1f9c98
WORKDIR /
COPY --from=builder /workspace/manager /manager
USER 65532:65532

ENTRYPOINT ["/manager"]

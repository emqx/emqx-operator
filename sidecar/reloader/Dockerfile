# Build the manager binary
FROM golang:1.18.3 AS builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go work init && go work use . && go mod download

# Copy the go source
COPY main.go main.go

RUN CGO_ENABLED=0 GOOS=linux go build -a -o reloader main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/reloader .
USER 65532:65532

ENTRYPOINT ["/reloader"]

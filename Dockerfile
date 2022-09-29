# Build the manager binary
FROM golang:1.18 as builder

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go sources
COPY main.go main.go
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a

# Use minimal base image to package the binary
FROM registry.access.redhat.com/ubi9-minimal
WORKDIR /
RUN microdnf install git jq -y && microdnf clean all
COPY --from=builder /workspace/copilot-ops .

# USER 65532:65532

ENTRYPOINT ["/copilot-ops"]

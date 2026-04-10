FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.25@sha256:bd531796aacb86e4f97443797262680fbf36ca048717c00b6f4248465e1a7c0c

WORKDIR /workspace

# Extract go version from the real go.mod and create a minimal one
COPY go.mod .
RUN grep '^go ' go.mod > /tmp/go_version.txt && \
    grep '^toolchain ' go.mod >> /tmp/go_version.txt || true

# Create minimal go.mod with only version info (no dependencies)
RUN echo "module github.com/stackrox/rox" > go.mod && \
    cat /tmp/go_version.txt >> go.mod

# Create a trivial Go file to validate compilation
RUN echo 'package main\n\nfunc main() {}' > main.go

# Validate Go version compatibility by building
# This will fail if go.mod requires a Go version higher than installed
RUN set -ex && \
    echo "Installed Go version:" && \
    go version && \
    echo "Minimal go.mod for validation:" && \
    cat go.mod && \
    echo "Checking go.mod compatibility..." && \
    go build -o /dev/null . && \
    echo "SUCCESS: Go version is compatible with go.mod"

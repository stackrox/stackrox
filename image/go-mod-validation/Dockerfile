FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.25@sha256:bd531796aacb86e4f97443797262680fbf36ca048717c00b6f4248465e1a7c0c

WORKDIR /workspace

# Copy only go.mod to check version compatibility
COPY go.mod .

# Validate Go version compatibility
# go mod tidy will fail if go.mod requires a Go version higher than installed
RUN set -ex && \
    echo "Installed Go version:" && \
    go version && \
    echo "go.mod version requirement:" && \
    grep -E '^(go|toolchain) ' go.mod && \
    echo "Checking go.mod compatibility..." && \
    go mod tidy && \
    echo "SUCCESS: Go version is compatible with go.mod"

# Test that go mod tidy actually fails on incompatible versions
# This validates we're not relying on behavior that silently changed
RUN set -ex && \
    echo "Testing go mod tidy failure detection..." && \
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//') && \
    echo "Current Go version: $GO_VERSION" && \
    echo "Testing with incompatible version: 2.0.0" && \
    mkdir -p /tmp/test-validation && \
    cd /tmp/test-validation && \
    echo "module test" > go.mod && \
    echo "go 2.0.0" >> go.mod && \
    echo "Created test go.mod:" && \
    cat go.mod && \
    if go mod tidy 2>&1; then \
        echo "ERROR: go mod tidy succeeded with incompatible version 2.0.0"; \
        echo "Our assumption about go mod tidy behavior is broken!"; \
        exit 1; \
    else \
        echo "SUCCESS: go mod tidy correctly failed with incompatible version 2.0.0"; \
    fi

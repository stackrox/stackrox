FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.25@sha256:bd531796aacb86e4f97443797262680fbf36ca048717c00b6f4248465e1a7c0c

WORKDIR /workspace

# Copy only go.mod to check version compatibility
COPY go.mod .

# Validate Go version compatibility
# go mod tidy will fail if go.mod requires a Go version higher than installed
RUN echo "Installed Go version:"
RUN go version
RUN echo "go.mod version requirement:"
RUN grep -E '^(go|toolchain) ' go.mod
RUN echo "Checking go.mod compatibility..."
RUN go mod tidy
RUN echo "SUCCESS: Go version is compatible with go.mod"

# Test that go mod tidy actually fails on incompatible versions
# This validates we're not relying on behavior that silently changed
RUN echo "Testing go mod tidy failure detection..."
RUN mkdir -p /tmp/test-validation
WORKDIR /tmp/test-validation
RUN echo "Testing with incompatible version: 2.0.0"
RUN echo "module test" > go.mod
RUN echo "go 2.0.0" >> go.mod
RUN echo "Created test go.mod:"
RUN cat go.mod
RUN if go mod tidy; then \
        echo "ERROR: go mod tidy succeeded with incompatible version 2.0.0"; \
        echo "Our assumption about go mod tidy behavior is broken!"; \
        exit 1; \
    else \
        echo "SUCCESS: go mod tidy correctly failed with incompatible version 2.0.0"; \
    fi

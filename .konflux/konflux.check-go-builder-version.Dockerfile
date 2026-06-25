FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_golang_1.26@sha256:aa2429f5dcf086b4edb2251f0cea19d54c60638aaedc03a1384d26ec3754cebc

WORKDIR /workspace

# Copy only go.mod to check version compatibility
COPY go.mod .

# Validate Go version compatibility
# go mod tidy will fail if go.mod requires a Go version higher than available in the builder
RUN echo "Go version of the builder:" && \
    go version 2>/dev/null
RUN echo "go.mod version requirement:" && \
    grep -E '^(go|toolchain) ' go.mod
RUN echo "If the following command fails, either roll back your changes to go.mod or update the openshift-golang-builder base images."
RUN echo "Checking go.mod compatibility..." && \
    go mod tidy
RUN echo "SUCCESS: Go version is compatible with go.mod"

# Test that go mod tidy actually fails on incompatible versions
# This validates we're not relying on behavior that silently changed
RUN echo "Testing go mod tidy failure detection..."
RUN go mod edit -go=1.200.0 2>/dev/null
RUN if go mod tidy; then \
        echo "ERROR: go mod tidy succeeded with incompatible version"; \
        echo "Our assumption about go mod tidy behavior is broken!"; \
        exit 1; \
    else \
        echo "SUCCESS: go mod tidy correctly detects an incompatible Go version"; \
    fi

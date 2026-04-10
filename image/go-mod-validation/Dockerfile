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

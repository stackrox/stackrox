FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.25@sha256:bd531796aacb86e4f97443797262680fbf36ca048717c00b6f4248465e1a7c0c

WORKDIR /workspace

# Copy source code
COPY . .

# Run go mod tidy to validate Go version compatibility
# This will fail if the Go version is incompatible with go.mod requirements
RUN go version && \
    go mod tidy && \
    echo "SUCCESS: Go version is compatible with go.mod"

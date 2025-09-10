FROM registry.redhat.io/ubi9/ubi-minimal:latest

LABEL name="StackRox Central Proxy" \
      vendor="Red Hat" \
      summary="Central Proxy for OpenShift Console Integration" \
      description="Provides cluster-local access to StackRox security data with OpenShift RBAC integration"

# Install ca-certificates for HTTPS requests
RUN microdnf update -y && \
    microdnf install -y ca-certificates && \
    microdnf clean all

# Create non-root user
RUN useradd -r -u 4000 central-proxy

# Copy the binary
COPY central-proxy /usr/local/bin/central-proxy

# Set ownership and permissions
RUN chown central-proxy:central-proxy /usr/local/bin/central-proxy && \
    chmod +x /usr/local/bin/central-proxy

# Switch to non-root user
USER 4000

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/central-proxy"]
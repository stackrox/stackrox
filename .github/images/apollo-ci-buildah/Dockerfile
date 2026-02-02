# Extends apollo-ci with buildah for daemonless container builds
FROM quay.io/stackrox-io/apollo-ci:stackrox-test-0.5.1

USER root

# Install buildah
RUN dnf install -y buildah && \
    dnf clean all && \
    rm -rf /var/cache/dnf

# Configure buildah for rootless operation
RUN mkdir -p /etc/containers && \
    echo '[storage]' > /etc/containers/storage.conf && \
    echo 'driver = "vfs"' >> /etc/containers/storage.conf

USER 1000

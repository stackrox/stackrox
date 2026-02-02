# Extends apollo-ci with buildah for daemonless container builds
FROM quay.io/stackrox-io/apollo-ci:stackrox-test-0.5.1

# Install buildah
RUN dnf install -y buildah && \
    dnf clean all && \
    rm -rf /var/cache/dnf

# Configure buildah storage with all required paths
RUN mkdir -p /etc/containers /var/lib/containers /run/containers && \
    echo '[storage]' > /etc/containers/storage.conf && \
    echo 'driver = "vfs"' >> /etc/containers/storage.conf && \
    echo 'runroot = "/run/containers/storage"' >> /etc/containers/storage.conf && \
    echo 'graphroot = "/var/lib/containers/storage"' >> /etc/containers/storage.conf

# Mark GitHub Actions workspace as safe for git
RUN git config --global --add safe.directory '*'

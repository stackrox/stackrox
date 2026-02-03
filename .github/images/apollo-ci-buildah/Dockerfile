# Multi-arch CI image for StackRox builds
# Version: 0.6.0
# Supports both amd64 and arm64 architectures
FROM registry.access.redhat.com/ubi9:latest

# This line makes sure that piped commands in RUN instructions exit early.
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Install base packages
RUN dnf install -y \
        buildah \
        git \
        make \
        gcc \
        gcc-c++ \
        unzip \
        wget \
        jq \
        which \
        procps-ng \
    && dnf clean all \
    && rm -rf /var/cache/dnf

# Install Go (multi-arch)
ARG GO_VERSION=1.24.0
RUN set -ex \
    && ARCH=$(uname -m) \
    && case "${ARCH}" in \
         x86_64) GOARCH=amd64 ;; \
         aarch64) GOARCH=arm64 ;; \
         *) echo "Unsupported architecture: ${ARCH}" && exit 1 ;; \
       esac \
    && wget --no-verbose "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" -O /tmp/go.tar.gz \
    && tar -C /usr/local -xzf /tmp/go.tar.gz \
    && rm /tmp/go.tar.gz \
    && /usr/local/go/bin/go version
ENV PATH="/usr/local/go/bin:/go/bin:${PATH}"
ENV GOPATH=/go

# Install Node.js (multi-arch)
ARG NODE_VERSION=20
RUN set -ex \
    && curl -fsSL https://rpm.nodesource.com/setup_${NODE_VERSION}.x | bash - \
    && dnf install -y nodejs \
    && dnf clean all \
    && rm -rf /var/cache/dnf \
    && node --version \
    && npm --version

# Configure buildah storage with all required paths
RUN mkdir -p /etc/containers /var/lib/containers/storage /run/containers/storage && \
    chmod -R 777 /var/lib/containers /run/containers && \
    echo '[storage]' > /etc/containers/storage.conf && \
    echo 'driver = "vfs"' >> /etc/containers/storage.conf && \
    echo 'runroot = "/run/containers/storage"' >> /etc/containers/storage.conf && \
    echo 'graphroot = "/var/lib/containers/storage"' >> /etc/containers/storage.conf

# Set buildah environment variables
ENV BUILDAH_ISOLATION=chroot
ENV STORAGE_DRIVER=vfs

# Mark GitHub Actions workspace as safe for git (system-wide)
RUN git config --system --add safe.directory '*'

# Circle CI uses BASH_ENV to pass an environment for bash
ENV BASH_ENV=/etc/initial-bash.env
RUN touch /etc/initial-bash.env

# Install protoc (multi-arch)
ARG PROTOC_VERSION=25.1
RUN set -ex \
    && ARCH=$(uname -m) \
    && case "${ARCH}" in \
         x86_64) PROTOC_ARCH=x86_64 ;; \
         aarch64) PROTOC_ARCH=aarch_64 ;; \
         *) echo "Unsupported architecture: ${ARCH}" && exit 1 ;; \
       esac \
    && wget --no-verbose "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-${PROTOC_ARCH}.zip" -O /tmp/protoc.zip \
    && unzip /tmp/protoc.zip -d /usr/local \
    && rm /tmp/protoc.zip \
    && protoc --version

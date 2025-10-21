# This Dockerfile uses a multi-stage build to create a minimal runtime image
# All package installations happen in builder stages (ubi8), then files are copied to ubi8-micro

ARG RPMS_REGISTRY=registry.access.redhat.com
ARG RPMS_BASE_IMAGE=ubi8
ARG RPMS_BASE_TAG=latest
ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8-micro
ARG BASE_TAG=latest
ARG PG_VERSION=15

# ============================================================================
# Stage 1: downloads - Download PostgreSQL RPMs and optionally build delve
# ============================================================================
FROM ${RPMS_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} AS downloads

ARG DEBUG_BUILD=no

WORKDIR /
COPY download.sh /download.sh
RUN /download.sh

# ============================================================================
# Stage 2: stackrox_data - Fetch external network definitions
# ============================================================================
FROM ${RPMS_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} AS stackrox_data

RUN mkdir /stackrox-data
# Install zip and openssl needed for fetch-stackrox-data.sh
RUN dnf upgrade --nobest -y && dnf install -y zip openssl && dnf clean all

WORKDIR /
COPY fetch-stackrox-data.sh .
RUN /fetch-stackrox-data.sh /stackrox-data

# ============================================================================
# Stage 3: dependency_builder - Install all runtime dependencies to /out/
# ============================================================================
FROM ${RPMS_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} AS dependency_builder

ARG PG_VERSION

# Install all required runtime dependencies to /out/ directory
# Using --installroot=/out/ means packages install to /out/ instead of /
# Combined with cleanup in single layer to reduce image size
RUN dnf install \
    --installroot=/out/ \
    --releasever=8 \
    --setopt=install_weak_deps=0 \
    --nodocs \
    -y \
    findutils \
    util-linux \
    ca-certificates \
    curl \
    bash \
    coreutils && \
    dnf --installroot=/out/ clean all && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum

# Install PostgreSQL RPMs from downloads stage to /out/
# Using rpm --root to install to alternate root
# Combined with cleanup to minimize layers
COPY --from=downloads /output/rpms/ /tmp/rpms/
RUN rpm --root=/out/ -ivh --nodeps /tmp/rpms/postgres-libs.rpm /tmp/rpms/postgres.rpm && \
    rm -rf /tmp/rpms

# Copy delve debugger if built (rarely changes, cache-friendly)
COPY --from=downloads /output/go/ /out/go/

# Copy static binaries (changes frequently during development)
COPY static-bin /out/stackrox/

# Import RPM GPG key (rarely changes)
COPY signatures/RPM-GPG-KEY-CentOS-Official /out/tmp/

# Setup directory structure and permissions in /out/
# The contents of paths mounted as emptyDir volumes in Kubernetes are saved
# by the script `save-dir-contents` during the image build. The directory
# contents are then restored by the script `restore-all-dir-contents`
# during the container start.
RUN mkdir -p /out/etc/pki/ca-trust/source/anchors /out/etc/ssl && \
    chown -R 4000:4000 /out/etc/pki/ca-trust /out/etc/ssl && \
    chroot /out /stackrox/save-dir-contents /etc/pki/ca-trust /etc/ssl && \
    mkdir -p /out/var/lib/stackrox /out/var/log/stackrox /out/var/cache/stackrox && \
    chown -R 4000:4000 /out/var/lib/stackrox /out/var/log/stackrox /out/var/cache/stackrox /out/tmp

# ============================================================================
# Stage 4: FINAL - ubi8-micro runtime image
# NO package manager operations allowed in this stage!
# ============================================================================
FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

ARG LABEL_VERSION
ARG LABEL_RELEASE
ARG QUAY_TAG_EXPIRATION
ARG ROX_IMAGE_FLAVOR
ARG ROX_PRODUCT_BRANDING
ARG TARGET_ARCH=amd64

LABEL name="main" \
      vendor="StackRox" \
      maintainer="https://stackrox.io/" \
      summary="The StackRox Kubernetes Security Platform" \
      description="This image contains components required to operate the StackRox Kubernetes Security Platform." \
      version="${LABEL_VERSION}" \
      release="${LABEL_RELEASE}" \
      quay.expires-after="${QUAY_TAG_EXPIRATION}"

ENV PATH="/stackrox:$PATH" \
    ROX_ROXCTL_IN_MAIN_IMAGE="true" \
    ROX_IMAGE_FLAVOR=${ROX_IMAGE_FLAVOR} \
    ROX_PRODUCT_BRANDING=${ROX_PRODUCT_BRANDING}

# Copy all dependencies from builder stage
COPY --from=dependency_builder /out/ /

# Copy stackrox data (external network definitions)
COPY --from=stackrox_data /stackrox-data /stackrox/static-data

# Copy API documentation
COPY ./docs/api/v1/swagger.json /stackrox/static-data/docs/api/v1/swagger.json
COPY ./docs/api/v2/swagger.json /stackrox/static-data/docs/api/v2/swagger.json

# Copy third-party notices
COPY THIRD_PARTY_NOTICES /THIRD_PARTY_NOTICES/

# Copy UI files
COPY ui /ui

# Copy application binaries
COPY bin/central            /stackrox/central
COPY bin/migrator           /stackrox/bin/migrator
COPY bin/compliance         /stackrox/bin/compliance
COPY bin/kubernetes-sensor  /stackrox/bin/kubernetes-sensor
COPY bin/sensor-upgrader    /stackrox/bin/sensor-upgrader
COPY bin/admission-control  /stackrox/bin/admission-control
COPY bin/config-controller  /stackrox/bin/config-controller
COPY bin/roxctl*            /assets/downloads/cli/

# Create symlinks for roxctl
RUN ln -s /assets/downloads/cli/roxctl-linux-${TARGET_ARCH} /stackrox/roxctl && \
    ln -s /assets/downloads/cli/roxctl-linux-amd64 /assets/downloads/cli/roxctl-linux

EXPOSE 8443

USER 4000:4000

ENTRYPOINT ["/stackrox/roxctl"]

HEALTHCHECK CMD curl --insecure --fail https://127.0.0.1:8443/v1/ping

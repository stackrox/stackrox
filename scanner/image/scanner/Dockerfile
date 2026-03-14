# Migration to ubi8-micro base image
# Scanner needs bash and utilities for entrypoint scripts

ARG MAPPINGS_REGISTRY=registry.access.redhat.com
ARG MAPPINGS_BASE_IMAGE=ubi8
ARG MAPPINGS_BASE_TAG=latest
ARG RPMS_REGISTRY=registry.access.redhat.com
ARG RPMS_BASE_IMAGE=ubi8
ARG RPMS_BASE_TAG=latest
ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8-micro
ARG BASE_TAG=latest

# ============================================================================
# Stage 1: mappings - Download vulnerability mappings
# ============================================================================
FROM ${MAPPINGS_REGISTRY}/${MAPPINGS_BASE_IMAGE}:${MAPPINGS_BASE_TAG} AS mappings

COPY download-mappings.sh /download-mappings.sh
RUN /download-mappings.sh /mappings

# ============================================================================
# Stage 2: dependency_builder - Install runtime dependencies for ubi8-micro
# ============================================================================
FROM ${RPMS_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} AS dependency_builder

# Install minimal required runtime dependencies to /out/
# These are needed for the shell scripts (entrypoint.sh, restore-all-dir-contents, etc.)
RUN dnf install \
    --installroot=/out/ \
    --releasever=8 \
    --setopt=install_weak_deps=0 \
    --nodocs \
    -y \
    bash \
    coreutils \
    findutils \
    util-linux \
    ca-certificates && \
    dnf --installroot=/out/ clean all && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum

# Copy scripts needed for CA trust setup
COPY scripts/save-dir-contents /out/usr/local/bin/

# Setup CA trust directories with proper permissions in /out/
RUN chown -R 65534:65534 /out/etc/pki/ca-trust /out/etc/ssl /out/tmp && \
    chroot /out /usr/local/bin/save-dir-contents /etc/pki/ca-trust /etc/ssl

# ============================================================================
# Stage 3: FINAL - ubi8-micro runtime image
# NO package manager operations allowed in this stage!
# ============================================================================
FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

ARG LABEL_VERSION
ARG LABEL_RELEASE
ARG QUAY_TAG_EXPIRATION

LABEL name="scanner-v4" \
      vendor="StackRox" \
      maintainer="https://stackrox.io/" \
      summary="Static vulnerability scanner for the StackRox Security Platform" \
      description="This image supports static vulnerability scanning for the StackRox Security Platform." \
      version="${LABEL_VERSION}" \
      release="${LABEL_RELEASE}" \
      quay.expires-after="${QUAY_TAG_EXPIRATION}"

# Copy all dependencies from builder (bash, coreutils, ca-certificates, etc.)
COPY --from=dependency_builder /out/ /

SHELL ["/bin/sh", "-o", "pipefail", "-c"]

# Copy application scripts and binaries
COPY scripts/entrypoint.sh \
     scripts/import-additional-cas \
     scripts/restore-all-dir-contents /usr/local/bin/
COPY bin/scanner /usr/local/bin/
COPY THIRD_PARTY_NOTICES/ /THIRD_PARTY_NOTICES/

# Copy vulnerability mappings
COPY --from=mappings /mappings/repository-to-cpe.json /mappings/container-name-repos-map.json /run/mappings/

# This is equivalent to nobody:nobody.
USER 65534:65534

ENTRYPOINT ["entrypoint.sh"]

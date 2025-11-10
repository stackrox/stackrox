ARG RPMS_REGISTRY=registry.access.redhat.com
ARG RPMS_BASE_IMAGE=ubi8
ARG RPMS_BASE_TAG=latest
ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8-minimal
ARG BASE_TAG=latest

FROM ${RPMS_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} AS downloads

ARG DEBUG_BUILD=no

WORKDIR /
COPY download.sh /download.sh
# Cache the download script separately from the execution
# This allows Docker to cache the download layer better
RUN --mount=type=cache,target=/var/cache/dnf \
    --mount=type=cache,target=/var/cache/yum \
    /download.sh

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG} AS stackrox_data

RUN mkdir /stackrox-data

# Install dependencies with cache mount
RUN --mount=type=cache,target=/var/cache/dnf \
    microdnf upgrade --nobest -y && microdnf install -y zip

WORKDIR /
COPY fetch-stackrox-data.sh .
# Use cache for external downloads
RUN --mount=type=cache,target=/tmp/stackrox-cache \
    /fetch-stackrox-data.sh /stackrox-data

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

COPY signatures/RPM-GPG-KEY-CentOS-Official /
COPY --from=downloads /output/rpms/ /tmp/
COPY --from=downloads /output/go/ /go/

# Copy static binaries first (needed for save-dir-contents script)
COPY static-bin /stackrox/

# Install system packages (most cacheable layer)
RUN --mount=type=cache,target=/var/cache/dnf \
    --mount=type=cache,target=/var/cache/yum \
    rpm --import RPM-GPG-KEY-CentOS-Official && \
    microdnf -y upgrade --nobest && \
    rpm -i --nodeps /tmp/postgres-libs.rpm && \
    rpm -i --nodeps /tmp/postgres.rpm && \
    microdnf install --setopt=install_weak_deps=0 --nodocs -y util-linux && \
    microdnf clean all -y && \
    rm /tmp/postgres.rpm /tmp/postgres-libs.rpm RPM-GPG-KEY-CentOS-Official && \
    rpm -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') || true

# Create directories and set permissions (separate layer for better caching)
RUN mkdir -p /var/lib/stackrox /var/log/stackrox /var/cache/stackrox && \
    chown -R 4000:4000 /etc/pki/ca-trust /etc/ssl /var/lib/stackrox /var/log/stackrox /var/cache/stackrox /tmp && \
    save-dir-contents /etc/pki/ca-trust /etc/ssl

# Copy static data (changes infrequently)
COPY --from=stackrox_data /stackrox-data /stackrox/static-data
COPY THIRD_PARTY_NOTICES /THIRD_PARTY_NOTICES/
COPY ./docs/api/v1/swagger.json /stackrox/static-data/docs/api/v1/swagger.json
COPY ./docs/api/v2/swagger.json /stackrox/static-data/docs/api/v2/swagger.json

# Copy UI (changes moderately)
COPY ui /ui

# Copy application binaries (changes most frequently - keep at end)
COPY bin/ /stackrox/bin/
# Move central to expected location
RUN mv /stackrox/bin/central /stackrox/central
# Create roxctl symlinks directory
RUN mkdir -p /assets/downloads/cli && \
    cp -p /stackrox/bin/roxctl* /assets/downloads/cli/ 2>/dev/null || true

RUN ln -s /assets/downloads/cli/roxctl-linux-${TARGET_ARCH} /stackrox/roxctl && \
    ln -s /assets/downloads/cli/roxctl-linux-amd64 /assets/downloads/cli/roxctl-linux

EXPOSE 8443

USER 4000:4000

ENTRYPOINT ["/stackrox/roxctl"]

HEALTHCHECK CMD curl --insecure --fail https://127.0.0.1:8443/v1/ping

ARG RPMS_REGISTRY=registry.access.redhat.com
ARG RPMS_BASE_IMAGE=ubi9
ARG RPMS_BASE_TAG=latest
ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi9-micro
ARG BASE_TAG=latest

FROM ${RPMS_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} AS ubi-base
FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG} AS ubi-micro-base
FROM ubi-base AS downloads

ARG DEBUG_BUILD=no

WORKDIR /
COPY download.sh /download.sh
RUN /download.sh

# Prepare stackrox-data using ubi9 (has zip utility)
FROM ubi-base AS stackrox_data

RUN dnf install -y zip

WORKDIR /
COPY fetch-stackrox-data.sh .
RUN mkdir /stackrox-data && /fetch-stackrox-data.sh /stackrox-data

FROM ubi-base AS package_installer

# Copy ubi-micro base to preserve rpmdb
COPY --from=ubi-micro-base / /out/

RUN dnf install -y \
        --installroot=/out/ \
        --releasever=9 \
        --setopt=install_weak_deps=0 \
        --nodocs \
        findutils \
        ca-certificates

# Copy PostgreSQL RPMs and install to /out/
COPY --from=downloads /output/rpms/ /tmp/
COPY signatures/RPM-GPG-KEY-CentOS-Official /tmp/
RUN rpm --import /tmp/RPM-GPG-KEY-CentOS-Official && \
    dnf install -y \
        --installroot=/out/ \
        --releasever=9 \
        --setopt=install_weak_deps=0 \
        --nodocs \
        /tmp/postgres-libs.rpm \
        /tmp/postgres.rpm && \
    dnf --installroot=/out/ clean all && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum

# Setup stackrox directories with correct ownership
RUN mkdir -p /out/stackrox && \
    mkdir -p /out/etc/pki/ca-trust/source/anchors /out/etc/ssl && \
    mkdir -p /out/var/lib/stackrox /out/var/log/stackrox /out/var/cache/stackrox && \
    chown -R 4000:4000 /out/etc/pki/ca-trust /out/etc/ssl /out/var/lib/stackrox /out/var/log/stackrox /out/var/cache/stackrox /out/tmp

COPY static-bin/* /out/stackrox/
RUN chroot /out /stackrox/save-dir-contents /etc/pki/ca-trust /etc/ssl

FROM ubi-micro-base

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

# Copy all dependencies and binaries from package_installer
COPY --from=package_installer /out/ /

COPY --from=downloads /output/go/ /go/

COPY --from=stackrox_data /stackrox-data /stackrox/static-data
COPY ./docs/api/v1/swagger.json /stackrox/static-data/docs/api/v1/swagger.json
COPY ./docs/api/v2/swagger.json /stackrox/static-data/docs/api/v2/swagger.json
COPY THIRD_PARTY_NOTICES /THIRD_PARTY_NOTICES/

COPY ui /ui

COPY bin/central            /stackrox/central
COPY bin/migrator           /stackrox/bin/migrator
COPY bin/compliance         /stackrox/bin/compliance
COPY bin/kubernetes-sensor  /stackrox/bin/kubernetes-sensor
COPY bin/sensor-upgrader    /stackrox/bin/sensor-upgrader
COPY bin/admission-control  /stackrox/bin/admission-control
COPY bin/config-controller  /stackrox/bin/config-controller
COPY bin/roxagent           /stackrox/bin/roxagent
COPY bin/roxctl*            /assets/downloads/cli/

RUN ln -s /assets/downloads/cli/roxctl-linux-${TARGET_ARCH} /stackrox/roxctl && \
    ln -s /assets/downloads/cli/roxctl-linux-amd64 /assets/downloads/cli/roxctl-linux

EXPOSE 8443

USER 4000:4000

ENTRYPOINT ["/stackrox/roxctl"]

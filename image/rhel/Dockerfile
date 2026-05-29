FROM registry.access.redhat.com/ubi9:latest AS ubi-base
FROM registry.access.redhat.com/ubi9/ubi-micro:latest AS ubi-micro-base

FROM ubi-base AS downloads

ARG DEBUG_BUILD=no

WORKDIR /
COPY --link download.sh /download.sh
RUN /download.sh

FROM ubi-base AS stackrox_data

RUN dnf install -y zip

WORKDIR /
COPY --link fetch-stackrox-data.sh .
RUN /fetch-stackrox-data.sh /stackrox-data

FROM ubi-base AS package_installer

COPY --link --from=ubi-micro-base / /out/
COPY --link --from=downloads /output/rpms/ /tmp/
COPY --link signatures/RPM-GPG-KEY-CentOS-Official /tmp/

RUN --mount=type=cache,target=/var/cache/dnf,sharing=shared \
    --mount=type=cache,target=/var/cache/yum,sharing=shared \
    rpm --import /tmp/RPM-GPG-KEY-CentOS-Official && \
    dnf install -y \
        --installroot=/out/ \
        --releasever=9 \
        --setopt=install_weak_deps=0 \
        --nodocs \
        findutils \
        ca-certificates \
        gzip \
        less \
        tar \
        /tmp/postgres-libs.rpm \
        /tmp/postgres.rpm && \
    dnf clean all --installroot=/out/

FROM ubi-micro-base

ARG TARGETARCH=amd64
ARG ROX_IMAGE_FLAVOR
ARG ROX_PRODUCT_BRANDING

ENV PATH="/stackrox:$PATH" \
    ROX_ROXCTL_IN_MAIN_IMAGE="true" \
    ROX_IMAGE_FLAVOR=${ROX_IMAGE_FLAVOR} \
    ROX_PRODUCT_BRANDING=${ROX_PRODUCT_BRANDING}

COPY --link --from=package_installer /out/ /

COPY --link static-bin /stackrox/
COPY --link --from=downloads /output/go/ /go/
COPY --link --from=stackrox_data /stackrox-data /stackrox/static-data
COPY --link ./docs/api/v1/swagger.json /stackrox/static-data/docs/api/v1/swagger.json
COPY --link ./docs/api/v2/swagger.json /stackrox/static-data/docs/api/v2/swagger.json
COPY --link THIRD_PARTY_NOTICES /THIRD_PARTY_NOTICES/

COPY --link ui /ui

COPY --link bin/central-${TARGETARCH}            /stackrox/central
COPY --link bin/migrator-${TARGETARCH}           /stackrox/bin/migrator
COPY --link bin/compliance-${TARGETARCH}         /stackrox/bin/compliance
COPY --link bin/kubernetes-${TARGETARCH}         /stackrox/bin/kubernetes-sensor
COPY --link bin/upgrader-${TARGETARCH}           /stackrox/bin/sensor-upgrader
COPY --link bin/admission-control-${TARGETARCH}  /stackrox/bin/admission-control
COPY --link bin/config-controller-${TARGETARCH}  /stackrox/bin/config-controller
COPY --link bin/roxagent-${TARGETARCH}           /stackrox/bin/roxagent
COPY --link bin/roxctl-*            /assets/downloads/cli/

RUN \
    # The contents of paths mounted as emptyDir volumes in Kubernetes are saved
    # by the script `save-dir-contents` during the image build. The directory
    # contents are then restored by the script `restore-all-dir-contents`
    # during the container start.
    chown -R 4000:4000 /etc/pki/ca-trust && save-dir-contents /etc/pki/ca-trust/source && \
    mkdir -p /var/lib/stackrox && chown -R 4000:4000 /var/lib/stackrox && \
    mkdir -p /var/log/stackrox && chown -R 4000:4000 /var/log/stackrox && \
    mkdir -p /var/cache/stackrox && chown -R 4000:4000 /var/cache/stackrox && \
    chown -R 4000:4000 /tmp && \
    ln -s /assets/downloads/cli/roxctl-linux-${TARGETARCH} /stackrox/roxctl && \
    ln -s /assets/downloads/cli/roxctl-linux-amd64 /assets/downloads/cli/roxctl-linux

ARG LABEL_VERSION
ARG LABEL_RELEASE
ARG QUAY_TAG_EXPIRATION

LABEL name="main" \
      vendor="StackRox" \
      maintainer="https://stackrox.io/" \
      summary="The StackRox Kubernetes Security Platform" \
      description="This image contains components required to operate the StackRox Kubernetes Security Platform." \
      version="${LABEL_VERSION}" \
      release="${LABEL_RELEASE}" \
      quay.expires-after="${QUAY_TAG_EXPIRATION}"

EXPOSE 8443

USER 4000:4000

ENTRYPOINT ["/stackrox/roxctl"]

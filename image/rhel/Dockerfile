FROM registry.access.redhat.com/ubi9:latest AS ubi-base
FROM registry.access.redhat.com/ubi9/ubi-micro:latest AS ubi-micro-base

FROM ubi-base AS downloads

ARG DEBUG_BUILD=no

WORKDIR /
COPY download.sh /download.sh
RUN /download.sh

FROM ubi-base AS stackrox_data

RUN dnf install -y zip

WORKDIR /
COPY fetch-stackrox-data.sh .
RUN /fetch-stackrox-data.sh /stackrox-data

FROM ubi-base AS package_installer

COPY --from=ubi-micro-base / /out/
COPY --from=downloads /output/rpms/ /tmp/
COPY signatures/RPM-GPG-KEY-CentOS-Official /tmp/

RUN rpm --import /tmp/RPM-GPG-KEY-CentOS-Official && \
    dnf install -y \
        --installroot=/out/ \
        --releasever=9 \
        --setopt=install_weak_deps=0 \
        --nodocs \
        findutils \
        ca-certificates \
        /tmp/postgres-libs.rpm \
        /tmp/postgres.rpm && \
    dnf clean all --installroot=/out/ && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum

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

COPY --from=package_installer /out/ /

COPY static-bin /stackrox/
COPY --from=downloads /output/go/ /go/

RUN \
    # The contents of paths mounted as emptyDir volumes in Kubernetes are saved
    # by the script `save-dir-contents` during the image build. The directory
    # contents are then restored by the script `restore-all-dir-contents`
    # during the container start.
    chown -R 4000:4000 /etc/pki/ca-trust && save-dir-contents /etc/pki/ca-trust/source && \
    mkdir -p /var/lib/stackrox && chown -R 4000:4000 /var/lib/stackrox && \
    mkdir -p /var/log/stackrox && chown -R 4000:4000 /var/log/stackrox && \
    mkdir -p /var/cache/stackrox && chown -R 4000:4000 /var/cache/stackrox && \
    chown -R 4000:4000 /tmp

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

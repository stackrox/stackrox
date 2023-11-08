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
RUN /download.sh

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG} as stackrox_data

RUN mkdir /stackrox-data
RUN microdnf upgrade --nobest -y  && microdnf install -y openssl zip

WORKDIR /
COPY fetch-stackrox-data.sh .
RUN /fetch-stackrox-data.sh

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
COPY static-bin /stackrox/

COPY --from=downloads /output/rpms/ /tmp/
COPY --from=downloads /output/go/ /go/

RUN if [[ "$(uname -m)" == "x86_64" ]]; then rpm -i /tmp/snappy.rpm && rm /tmp/snappy.rpm; fi
RUN rpm --import RPM-GPG-KEY-CentOS-Official && \
    microdnf -y upgrade --nobest && \
    rpm -i --nodeps /tmp/postgres-libs.rpm && \
    rpm -i --nodeps /tmp/postgres.rpm && \
    microdnf install --setopt=install_weak_deps=0 --nodocs -y lz4 bzip2 util-linux && \
    microdnf clean all -y && \
    rm /tmp/postgres.rpm /tmp/postgres-libs.rpm RPM-GPG-KEY-CentOS-Official && \
    # (Optional) Remove line below to keep package management utilities
    # TODO(ROX-13934): Potentially add *RPM* to this list again
    rpm -e --nodeps $(rpm -qa curl '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum && \
    # The contents of paths mounted as emptyDir volumes in Kubernetes are saved
    # by the script `save-dir-contents` during the image build. The directory
    # contents are then restored by the script `restore-all-dir-contents`
    # during the container start.
    chown -R 4000:4000 /etc/pki /etc/ssl && save-dir-contents /etc/pki/ca-trust /etc/ssl && \
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
COPY bin/roxctl*            /assets/downloads/cli/

RUN ln -s /assets/downloads/cli/roxctl-linux-${TARGET_ARCH} /stackrox/roxctl && \
    ln -s /assets/downloads/cli/roxctl-linux-amd64 /assets/downloads/cli/roxctl-linux

EXPOSE 8443

USER 4000:4000

ENTRYPOINT ["/stackrox/roxctl"]

HEALTHCHECK CMD curl --insecure --fail https://127.0.0.1:8443/v1/ping

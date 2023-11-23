ARG RPMS_REGISTRY=registry.access.redhat.com
ARG RPMS_BASE_IMAGE=ubi8
ARG RPMS_BASE_TAG=latest
ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8-micro
ARG BASE_TAG=latest

FROM ${RPMS_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} AS downloads

ARG DEBUG_BUILD=no

WORKDIR /
COPY download.sh /download.sh
RUN /download.sh

FROM ${BASE_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} as stackrox_data

RUN mkdir /stackrox-data
RUN dnf upgrade --nobest -y  && dnf install -y openssl zip

WORKDIR /
COPY fetch-stackrox-data.sh .
RUN /fetch-stackrox-data.sh

FROM ${BASE_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} as builder

COPY signatures/RPM-GPG-KEY-CentOS-Official /
COPY static-bin /stackrox/

COPY --from=downloads /output/rpms/ /tmp/
COPY --from=downloads /output/go/ /go/

RUN mkdir out

#RUN if [[ "$(uname -m)" == "x86_64" ]]; then rpm --prefix=/out/ -ivh /tmp/snappy.rpm && rm /tmp/snappy.rpm; fi
#RUN dnf install -y  https://download.postgresql.org/pub/repos/yum/reporpms/EL-7-x86_64/pgdg-redhat-repo-latest.noarch.rpm
#RUN rm -rf /var/cache/dnf
#RUN dnf update --refresh && dnf -y  upgrade --nobest
#RUN dnf install --installroot=/out/ --setopt=install_weak_deps=0 --nodocs -y postgresql16-server
#RUN    rpm  --prefix=/out/ -ivh --nodeps /tmp/postgres-libs.rpm
#RUN    rpm  --prefix=/out/ -ivh --nodeps /tmp/postgres.rpm
RUN    dnf install --installroot=/out/ --setopt=install_weak_deps=0 --nodocs -y lz4 bzip2 util-linux
RUN    dnf clean all -y
RUN    rm /tmp/postgres.rpm /tmp/postgres-libs.rpm RPM-GPG-KEY-CentOS-Official
RUN    # (Optional) Remove line below to keep package management utilities
    # TODO(ROX-13934): Potentially add *RPM* to this list again
RUN    rpm -e --nodeps $(rpm -qa curl '*dnf*' '*libsolv*' '*hawkey*' 'yum*')
RUN    rm -rf /var/cache/dnf /var/cache/yum
RUN    # The contents of paths mounted as emptyDir volumes in Kubernetes are saved
    # by the script `save-dir-contents` during the image build. The directory
    # contents are then restored by the script `restore-all-dir-contents`
    # during the container start.
ENV PATH="/stackrox:$PATH"
RUN    chown -R 4000:4000 /etc/pki /etc/ssl && save-dir-contents /etc/pki/ca-trust /etc/ssl
RUN    mkdir -p /var/lib/stackrox && chown -R 4000:4000 /var/lib/stackrox
RUN    mkdir -p /var/log/stackrox && chown -R 4000:4000 /var/log/stackrox
RUN    mkdir -p /var/cache/stackrox && chown -R 4000:4000 /var/cache/stackrox
RUN    chown -R 4000:4000 /tmp

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

ARG LABEL_VERSION
ARG LABEL_RELEASE
ARG QUAY_TAG_EXPIRATION
ARG ROX_IMAGE_FLAVOR
ARG ROX_PRODUCT_BRANDING
ARG TARGET_ARCH=amd64

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

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

COPY --from=builder /out /
COPY --from=builder /ui /ui
COPY --from=builder /stackrox /stackrox/
COPY --from=builder /assets /assets/

RUN ln -s /assets/downloads/cli/roxctl-linux-${TARGET_ARCH} /stackrox/roxctl
RUN    ln -s /assets/downloads/cli/roxctl-linux-amd64 /assets/downloads/cli/roxctl-linux

EXPOSE 8443

USER 4000:4000

ENTRYPOINT ["/stackrox/roxctl"]

HEALTHCHECK CMD curl --insecure --fail https://127.0.0.1:8443/v1/ping

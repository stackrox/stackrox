ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8/ubi
ARG BASE_TAG=8.2

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

LABEL name="main-rhel" \
      vendor="StackRox" \
      maintainer="support@stackrox.com" \
      summary="The StackRox Kubernetes Security Platform" \
      description="This image contains components required to operate the StackRox Kubernetes Security Platform."

ENV PATH="/stackrox:$PATH" \
    ROX_ROXCTL_IN_MAIN_IMAGE="true"

COPY signatures/RPM-GPG-KEY-CentOS-Official /
COPY scripts /stackrox/
COPY bundle.tar.gz /

RUN ln -s entrypoint-wrapper.sh /stackrox/admission-control && \
    ln -s entrypoint-wrapper.sh /stackrox/compliance && \
    ln -s entrypoint-wrapper.sh /stackrox/kubernetes-sensor && \
    ln -s entrypoint-wrapper.sh /stackrox/sensor-upgrader && \
    tar -zxf bundle.tar.gz ./assets/ && \
    ln -s /assets/downloads/cli/roxctl-linux /stackrox/roxctl && \
    tar -zxf bundle.tar.gz ./snappy.rpm && \
    tar -zxf bundle.tar.gz ./stackrox/central && \
    tar -zxf bundle.tar.gz ./stackrox/bin/ && \
    tar -zxf bundle.tar.gz ./stackrox/stackrox-data.tgze && \
    tar -zxf bundle.tar.gz ./THIRD_PARTY_NOTICES/ && \
    tar -zxf bundle.tar.gz ./ui/ && \
    tar -zxf bundle.tar.gz ./usr/local/bin/ldb && \
    rm -f bundle.tar.gz && \
    rpm --import RPM-GPG-KEY-CentOS-Official && \
    dnf upgrade -y && \
    dnf install -y lz4 bzip2 snappy.rpm && \
    dnf clean all && \
    rm -rf /var/cache/dnf && \
    rm snappy.rpm RPM-GPG-KEY-CentOS-Official && \
    # (Optional) Remove line below to keep package management utilities
    rpm -e --nodeps rpm rpm-build-libs rpm-libs python3-rpm subscription-manager python3-subscription-manager-rhsm yum $(rpm -qa *dnf*) python3-hawkey && \
    # The contents of paths mounted as emptyDir volumes in Kubernetes are saved
    # by the script `save-dir-contents` during the image build. The directory
    # contents are then restored by the script `restore-all-dir-contents`
    # during the container start.
    chown -R 4000:4000 /etc/pki /etc/ssl && save-dir-contents /etc/pki/ca-trust /etc/ssl && \
    mkdir -p /var/lib/stackrox && chown -R 4000:4000 /var/lib/stackrox && \
    mkdir -p /var/log/stackrox && chown -R 4000:4000 /var/log/stackrox && \
    mkdir -p /var/cache/stackrox && chown -R 4000:4000 /var/cache/stackrox && \
    chown -R 4000:4000 /tmp && \
    mkdir -p /stackrox/data && chown -R 4000:4000 /stackrox/data

EXPOSE 8443

USER 4000:4000

ENTRYPOINT ["/assets/downloads/cli/roxctl-linux"]

HEALTHCHECK CMD curl --insecure --fail https://127.0.0.1:8443/v1/ping

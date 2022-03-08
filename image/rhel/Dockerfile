ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8-micro
ARG BASE_TAG=8.5

FROM ${BASE_REGISTRY}/ubi8/ubi:${BASE_TAG} AS extracted_bundle

COPY bundle.tar.gz /
WORKDIR /bundle
RUN tar -zxf /bundle.tar.gz
RUN dnf install -y lz4-libs bzip2-libs /bundle/snappy.rpm --nodocs

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

LABEL name="main" \
      vendor="StackRox" \
      maintainer="support@stackrox.com" \
      summary="The StackRox Kubernetes Security Platform" \
      description="This image contains components required to operate the StackRox Kubernetes Security Platform."

ARG ROX_IMAGE_FLAVOR

ENV PATH="/stackrox:$PATH" \
    ROX_ROXCTL_IN_MAIN_IMAGE="true" \
    ROX_IMAGE_FLAVOR=${ROX_IMAGE_FLAVOR}

COPY signatures/RPM-GPG-KEY-CentOS-Official /
COPY scripts /stackrox/
COPY --from=extracted_bundle /bundle/assets/ /assets/
COPY --from=extracted_bundle /bundle/stackrox/ /stackrox/
COPY --from=extracted_bundle /bundle/THIRD_PARTY_NOTICES/ /THIRD_PARTY_NOTICES/
COPY --from=extracted_bundle /bundle/ui/ /ui/
COPY --from=extracted_bundle /bundle/usr/local/bin/ldb /usr/local/bin/
COPY --from=extracted_bundle /bundle/go/ /go/
COPY --from=extracted_bundle /usr/lib64/liblz4.so.1 /usr/lib64/liblz4.so.1
COPY --from=extracted_bundle /usr/lib64/libbz2.so.1 /usr/lib64/libbz2.so.1
COPY --from=extracted_bundle /usr/lib64/libsnappy.so.1 /usr/lib64/libsnappy.so.1
COPY --from=extracted_bundle /usr/lib64/libstdc++.so.6 /usr/lib64/libstdc++.so.6
COPY --from=extracted_bundle /usr/lib64/libzstd.so.1 /usr/lib64/libzstd.so.1
COPY --from=extracted_bundle /usr/lib64/libz.so.1 /usr/lib64/libz.so.1

RUN ln -s entrypoint-wrapper.sh /stackrox/admission-control && \
    ln -s entrypoint-wrapper.sh /stackrox/compliance && \
    ln -s entrypoint-wrapper.sh /stackrox/kubernetes-sensor && \
    ln -s entrypoint-wrapper.sh /stackrox/sensor-upgrader && \
    ln -s /assets/downloads/cli/roxctl-linux /stackrox/roxctl && \
    mkdir -p /var/lib/stackrox && chown -R 4000:4000 /var/lib/stackrox && \
    mkdir -p /var/log/stackrox && chown -R 4000:4000 /var/log/stackrox && \
    mkdir -p /var/cache/stackrox && chown -R 4000:4000 /var/cache/stackrox && \
    mkdir -p /etc/pki /etc/ssl && \
    chown -R 4000:4000 /tmp /etc/pki /etc/ssl

EXPOSE 8443

USER 4000:4000

ENTRYPOINT ["/assets/downloads/cli/roxctl-linux"]

HEALTHCHECK CMD curl --insecure --fail https://127.0.0.1:8443/v1/ping

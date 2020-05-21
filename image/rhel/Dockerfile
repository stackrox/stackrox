ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8/ubi
ARG BASE_TAG=8.2

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

LABEL name="stackrox.io/main"
LABEL vendor="StackRox (support@stackrox.com)"
LABEL summary="The StackRox Kubernetes Security Platform"
LABEL description="This image contains everything required to operate the StackRox Kubernetes Security Platform."

ADD bundle.tar.gz /

RUN yum update -y --nogpgcheck --disableplugin=subscription-manager && \
    yum install -y --nogpgcheck --disableplugin=subscription-manager \
     lz4.rpm \
     bzip2.rpm \
     snappy.rpm \
     zlib.rpm

RUN rpm -e --nodeps \
      rpm \
      rpm-build-libs \
      rpm-libs \
      python3-rpm \
      subscription-manager \
      python3-subscription-manager-rhsm \
      yum \
      $(rpm -qa *dnf*) \
      python3-hawkey \
    ;

EXPOSE 8443

ENTRYPOINT ["/assets/downloads/cli/roxctl-linux"]

ENV PATH="/stackrox:$PATH"

ENV ROX_ROXCTL_IN_MAIN_IMAGE="true"

RUN chown -R 4000:4000 /etc/pki /etc/ssl && save-dir-contents /etc/pki/ca-trust /etc/ssl

VOLUME /etc/pki/ca-trust
VOLUME /etc/ssl

RUN mkdir -p /var/lib/stackrox && chown -R 4000:4000 /var/lib/stackrox
VOLUME /var/lib/stackrox

RUN mkdir -p /var/log/stackrox && chown -R 4000:4000 /var/log/stackrox
VOLUME /var/log/stackrox

RUN chown -R 4000:4000 /tmp
VOLUME /tmp

RUN mkdir -p /stackrox/data && chown -R 4000:4000 /stackrox/data
VOLUME /stackrox/data

USER 4000:4000

HEALTHCHECK CMD curl --insecure --fail https://127.0.0.1:8443/v1/ping

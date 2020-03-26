FROM alpine:3.10
ARG ALPINE_MIRROR=sjc.edge.kernel.org
ARG DOCS_BUNDLE_VERSION

RUN mkdir /stackrox-data

RUN wget -O product-docs.tgz https://storage.googleapis.com/doc-bundles/03c318a8759d13e8ed7611bccd6618dde60d768a345ff3c0a870e60c53bcfbe9/$DOCS_BUNDLE_VERSION.tgz && \
    tar xzf product-docs.tgz && \
    mv public /stackrox-data/product-docs && \
    ls /stackrox-data/product-docs/index.html && \
    rm product-docs.tgz

RUN echo http://$ALPINE_MIRROR/alpine/v3.10/main > /etc/apk/repositories; \
    echo http://$ALPINE_MIRROR/alpine/v3.10/community >> /etc/apk/repositories

RUN apk update && \
    apk add --no-cache \
        openssl \
        && \
    apk --purge del apk-tools \
    ;

RUN mkdir -p /stackrox-data/cve/k8s && \
    wget -O /stackrox-data/cve/k8s/checksum "https://definitions.stackrox.io/cve/k8s/checksum" && \
    wget -O /stackrox-data/cve/k8s/cve-list.json "https://definitions.stackrox.io/cve/k8s/cve-list.json" && \
    mkdir -p /stackrox-data/cve/istio && \
    wget -O /stackrox-data/cve/istio/checksum "https://definitions.stackrox.io/cve/istio/checksum" && \
    wget -O /stackrox-data/cve/istio/cve-list.json "https://definitions.stackrox.io/cve/istio/cve-list.json"

COPY ./policies/files /stackrox-data/policies/files
COPY ./docs/api/v1/swagger.json /stackrox-data/docs/api/v1/swagger.json

COPY ./keys /tmp/keys

RUN set -eo pipefail; \
	( cd /stackrox-data ; tar -czf - * ; ) | \
    openssl enc -aes-256-cbc \
        -K "$(hexdump -e '32/1 "%02x"' </tmp/keys/data-key)" \
        -iv "$(hexdump -e '16/1 "%02x"' </tmp/keys/data-iv)" \
        -out /stackrox-data.tgze

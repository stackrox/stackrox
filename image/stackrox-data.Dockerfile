ARG DOCS_IMAGE

FROM $DOCS_IMAGE AS docs

# Make sure docs data is in the right place.
# Note: we copy this to a well-known location in order to not depend on the
# WORKDIR of the docs image. The WORKDIR cannot be referenced as via
# `COPY --from=docs ./`, as in this context, ./ will refer to the container
# root.
USER 0
RUN mkdir -p /docs/public
RUN cp -a . /docs/public

FROM alpine:3.14

RUN mkdir /stackrox-data

RUN apk update && \
    apk add --no-cache \
        openssl zip \
        && \
    apk --purge del apk-tools \
    ;

COPY --from=docs /docs/public/ /stackrox-data/product-docs/
# Basic sanity check: are the docs in the right place?
RUN ls /stackrox-data/product-docs/index.html

RUN mkdir -p /stackrox-data/cve/istio && \
    wget -O /stackrox-data/cve/istio/checksum "https://definitions.stackrox.io/cve/istio/checksum" && \
    wget -O /stackrox-data/cve/istio/cve-list.json "https://definitions.stackrox.io/cve/istio/cve-list.json"

RUN mkdir -p /tmp/external-networks && \
    latest_prefix="$(wget -q https://definitions.stackrox.io/external-networks/latest_prefix -O -)" && \
    wget -O /tmp/external-networks/checksum "https://definitions.stackrox.io/${latest_prefix}/checksum" && \
    wget -O /tmp/external-networks/networks "https://definitions.stackrox.io/${latest_prefix}/networks" && \
    test -s /tmp/external-networks/checksum && test -s /tmp/external-networks/networks && \
    mkdir /stackrox-data/external-networks && \
    zip -jr /stackrox-data/external-networks/external-networks.zip /tmp/external-networks && \
    rm -rf /tmp/external-networks

COPY ./docs/api/v1/swagger.json /stackrox-data/docs/api/v1/swagger.json

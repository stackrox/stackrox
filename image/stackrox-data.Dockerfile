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

COPY fetch-stackrox-data.sh .
RUN sh -x fetch-stackrox-data.sh && \
    rm fetch-stackrox-data.sh

COPY ./docs/api/v1/swagger.json /stackrox-data/docs/api/v1/swagger.json

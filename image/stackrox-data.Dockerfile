FROM alpine:3.9

RUN mkdir /stackrox-data

RUN wget -O product-docs.tgz https://storage.googleapis.com/doc-bundles/03c318a8759d13e8ed7611bccd6618dde60d768a345ff3c0a870e60c53bcfbe9/0.0.0-82-gc16cd000.tgz && \
    tar xzf product-docs.tgz && \
    mv public /stackrox-data/product-docs && \
    ls /stackrox-data/product-docs/index.html && \
    rm product-docs.tgz

RUN apk update && \
    apk add --no-cache \
        openssl=1.1.1b-r1 \
        && \
    apk --purge del apk-tools \
    ;


COPY ./policies/files /stackrox-data/policies/files
COPY ./docs/api/v1/swagger.json /stackrox-data/docs/api/v1/swagger.json

COPY ./keys /tmp/keys

RUN set -eo pipefail; \
	tar -C /stackrox-data -czf - . | \
    openssl enc -aes-256-cbc \
        -K "$(hexdump -e '32/1 "%02x"' </tmp/keys/data-key)" \
        -iv "$(hexdump -e '16/1 "%02x"' </tmp/keys/data-iv)" \
        -out /stackrox-data.tgze

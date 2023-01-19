FROM alpine:3.14

RUN mkdir /stackrox-data

RUN apk update && \
    apk add --no-cache \
        openssl zip \
        && \
    apk --purge del apk-tools \
    ;

COPY fetch-stackrox-data.sh .
RUN sh -x fetch-stackrox-data.sh && \
    rm fetch-stackrox-data.sh

COPY ./docs/api/v1/swagger.json /stackrox-data/docs/api/v1/swagger.json

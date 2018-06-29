FROM alpine:3.7
MAINTAINER StackRox <support@stackrox.com>

RUN printf "http://apk.rox.systems/v3.7/main\nhttp://apk.rox.systems/v3.7/community" > /etc/apk/repositories

RUN apk update && \
    apk add \
        ca-certificates \
        procps \
        && \
    apk --purge del apk-tools \
    ;

ENV PATH=/prevent

COPY ./benchmarks/ /data/benchmarks
COPY ./policies/ /data/policies
COPY ./bin/ /prevent/
COPY ./ui/build  /ui/
COPY ./VERSION /VERSION

EXPOSE 443

ENTRYPOINT ["deploy"]

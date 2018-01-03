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

ENV PATH=/apollo

ADD ./benchmarks/ /data/benchmarks
ADD ./policies/ /data/policies
ADD ./bin/ /apollo/
ADD ./ui/build  /ui/

EXPOSE 443

ENTRYPOINT ["apollo"]

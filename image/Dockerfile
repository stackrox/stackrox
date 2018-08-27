FROM alpine:3.8
LABEL "com.vendor"="StackRox (support@stackrox.com)"

RUN printf "http://apk.rox.systems/v3.8/main\\nhttp://apk.rox.systems/v3.8/community" > /etc/apk/repositories

RUN apk update && \
    apk add --no-cache \
        ca-certificates=20171114-r3 \
        procps=3.3.15-r0 \
        && \
    apk --purge del apk-tools \
    ;

ENV PATH="/prevent:$PATH"

COPY ./benchmarks/ /data/benchmarks
COPY ./policies/ /data/policies
COPY ./bin/ /prevent/
COPY ./ui/build  /ui/
COPY ./VERSION /VERSION

EXPOSE 443

ENTRYPOINT ["deploy"]

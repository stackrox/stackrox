FROM alpine:3.8
LABEL "com.vendor"="StackRox (support@stackrox.com)"

RUN printf "http://apk.rox.systems/v3.8/main\\nhttp://apk.rox.systems/v3.8/community" > /etc/apk/repositories

# Pulled from https://github.com/connorgorman/telegraf/releases with the slimmed down version of telegraf
RUN wget https://github.com/connorgorman/telegraf/releases/download/1.8.0%2B49-slim/telegraf -O /telegraf && chmod +x /telegraf

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
COPY ./templates/ /data/templates
COPY ./entrypoint.sh /prevent/
COPY ./bin/benchmark-bootstrap /prevent/
COPY ./bin/benchmarks /prevent/
COPY ./bin/central /prevent/
COPY ./bin/deploy /prevent/
COPY ./bin/kubernetes-sensor /prevent/
COPY ./bin/swarm-sensor /prevent/
COPY ./ui/build  /ui/
COPY ./VERSION /VERSION

COPY ./telegraf.sh /telegraf.sh
COPY ./assets /data/assets

EXPOSE 443

ENTRYPOINT ["deploy"]

# The following paths are written to in Central.
# Including these VOLUME instructions means that we can enable a
# read-only root file system without preventing writes to these paths.
# Trusted CA certificates are written to this path by `update-ca-certificates`.
VOLUME /etc/ssl
# Search indices are generated upon startup and updated thereafter.
VOLUME /tmp/search
# The BoltDB is stored at this path.
VOLUME /var/lib/prevent
# The log file is stored at this path
VOLUME /var/log/stackrox

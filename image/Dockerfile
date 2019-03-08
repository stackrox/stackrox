FROM alpine:3.9
LABEL "com.vendor"="StackRox (support@stackrox.com)"

RUN printf "http://apk.rox.systems/v3.9/main\\nhttp://apk.rox.systems/v3.9/community" > /etc/apk/repositories

# Pulled from https://github.com/connorgorman/telegraf/releases with the slimmed down version of telegraf
RUN wget https://github.com/connorgorman/telegraf/releases/download/1.8.0%2B49-slim/telegraf -O /telegraf && chmod +x /telegraf

RUN apk update && \
    apk add --no-cache \
        ca-certificates=20190108-r0 \
        procps=3.3.15-r0 \
        && \
    apk --purge del apk-tools \
    ;

ENV PATH="/stackrox:$PATH"

COPY ./policies/ /data/policies
COPY ./templates/ /data/templates
COPY ./central-entrypoint.sh /stackrox/
COPY ./bin/migrator /stackrox/
COPY ./bin/central /stackrox/
COPY ./bin/compliance /stackrox/compliance
COPY ./bin/roxctl* /assets/downloads/cli/
COPY ./bin/kubernetes-sensor /stackrox/
COPY ./static-bin/* /stackrox/
COPY ./ui/build  /ui/
COPY ./docs/api/v1/swagger.json /docs/api/v1/swagger.json

COPY ./NOTICE.txt /

COPY ./assets /data/assets

EXPOSE 443

ENTRYPOINT ["/assets/downloads/cli/roxctl-linux"]

# The following paths are written to in Central.
# Including these VOLUME instructions means that we can enable a
# read-only root file system without preventing writes to these paths.
# Trusted CA certificates are written to this path by `update-ca-certificates`.
VOLUME /etc/ssl
# Search indices are generated upon startup and updated thereafter.
VOLUME /tmp/search
# The BoltDB is stored at this path.
VOLUME /var/lib/stackrox
# The log file is stored at this path
VOLUME /var/log/stackrox

# /tmp needs to be a read-write filesystem.
VOLUME /tmp

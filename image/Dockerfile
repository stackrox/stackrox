FROM alpine:3.9
LABEL "com.vendor"="StackRox (support@stackrox.com)"

RUN printf "http://apk.rox.systems/v3.9/main\\nhttp://apk.rox.systems/v3.9/community" > /etc/apk/repositories

# Pulled from https://github.com/connorgorman/telegraf/releases with the slimmed down version of telegraf
RUN wget https://github.com/connorgorman/telegraf/releases/download/1.8.0%2B49-slim/telegraf -O /telegraf && chmod +x /telegraf

RUN apk update && \
    apk add --no-cache \
        ca-certificates=20190108-r0 \
        procps=3.3.15-r0 \
        su-exec=0.2-r0 \
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

EXPOSE 8443

ENTRYPOINT ["/assets/downloads/cli/roxctl-linux"]

# The following paths are written to in Central.
# Including these VOLUME instructions means that we can enable a
# read-only root file system without preventing writes to these paths.
# Trusted CA certificates are written to this path by `update-ca-certificates`.
RUN chown -R 4000 /etc/ssl
VOLUME /etc/ssl
# Search indices are generated upon startup and updated thereafter.
RUN mkdir -p /tmp/search && chown -R 4000 /tmp/search
VOLUME /tmp/search
# The BoltDB is stored at this path.
RUN mkdir -p /var/lib/stackrox && chown -R 4000 /var/lib/stackrox
VOLUME /var/lib/stackrox
# The log file is stored at this path
RUN mkdir -p /var/log/stackrox && chown -R 4000 /var/log/stackrox
VOLUME /var/log/stackrox

# /tmp needs to be a read-write filesystem.
RUN chown -R 4000 /tmp
VOLUME /tmp

FROM alpine:3.6
MAINTAINER StackRox <support@stackrox.com>

RUN printf "http://apk.rox.systems/v3.6/main\nhttp://apk.rox.systems/v3.6/community" > /etc/apk/repositories

RUN apk update && apk add ca-certificates lvm2-dev btrfs-progs-dev procps && apk --purge del apk-tools
#RUN apk update && apk add  alpine-sdk  && apk --purge del apk-tools
ENV PATH=/

COPY ./bin/docker-bench /docker-bench

HEALTHCHECK CMD exit 0

ENTRYPOINT ["/docker-bench"]

FROM alpine:3.6
MAINTAINER StackRox <support@stackrox.com>

RUN printf "http://apk.rox.systems/v3.6/main\nhttp://apk.rox.systems/v3.6/community" > /etc/apk/repositories

RUN apk update && apk add ca-certificates procps && apk --purge del apk-tools

ENV PATH=/

ADD ./bin/apollo /apollo

EXPOSE 8080

ENTRYPOINT ["apollo"]

FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY ./bin/roxctl-linux /roxctl
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENV ROX_ROXCTL_IN_MAIN_IMAGE="true"

USER 65534:65534
ENTRYPOINT [ "/roxctl" ]

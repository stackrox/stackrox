FROM registry.access.redhat.com/ubi8-minimal:latest AS certs

FROM scratch
COPY ./roxctl-linux /roxctl
COPY --from=certs /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem /etc/ssl/certs/ca-certificates.crt
ENV ROX_ROXCTL_IN_MAIN_IMAGE="true"

USER 65534:65534
ENTRYPOINT [ "/roxctl" ]

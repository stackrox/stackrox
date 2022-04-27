# Under OpenShift CI CA certificates are in context already.
FROM scratch
COPY ./bin/roxctl-linux /roxctl
COPY ./tls-ca-bundle.pem /etc/ssl/certs/ca-certificates.crt
ENV ROX_ROXCTL_IN_MAIN_IMAGE="true"

USER 65534:65534
ENTRYPOINT [ "/roxctl" ]

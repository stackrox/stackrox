ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8-minimal
ARG BASE_TAG=8.5

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG} AS certs

FROM scratch
COPY ./bin/roxctl-linux /roxctl
COPY --from=certs /etc/ssl/certs/ca-bundle.crt /etc/ssl/certs/ca-certificates.crt
ENV ROX_ROXCTL_IN_MAIN_IMAGE="true"

USER 65534:65534
ENTRYPOINT [ "/roxctl" ]

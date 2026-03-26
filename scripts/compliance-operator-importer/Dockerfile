FROM registry.access.redhat.com/ubi9-micro:latest

LABEL org.opencontainers.image.title="co-acs-importer" \
      org.opencontainers.image.description="Compliance Operator to ACS scan configuration importer" \
      org.opencontainers.image.source="https://github.com/stackrox/stackrox"

COPY compliance-operator-importer /compliance-operator-importer

USER 65534:65534
ENTRYPOINT ["/compliance-operator-importer"]

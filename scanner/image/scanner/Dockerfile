ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8-minimal
ARG BASE_TAG=latest

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

LABEL name="scanner" \
      vendor="StackRox" \
      maintainer="https://stackrox.io/" \
      summary="Static vulnerability scanner for the StackRox Security Platform" \
      description="This image supports static vulnerability scanning for the StackRox Security Platform."

SHELL ["/bin/sh", "-o", "pipefail", "-c"]

COPY scripts/entrypoint.sh \
     scripts/import-additional-cas \
     scripts/restore-all-dir-contents \
     scripts/save-dir-contents /usr/local/bin/
COPY bin/scanner /usr/local/bin/
COPY THIRD_PARTY_NOTICES/ /THIRD_PARTY_NOTICES/

RUN microdnf upgrade --nobest && \
    microdnf clean all && \
    # (Optional) Remove line below to keep package management utilities
    rpm -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum && \
    chown -R 65534:65534 /tmp && \
    # The contents of paths mounted as emptyDir volumes in Kubernetes are saved
    # by the script `save-dir-contents` during the image build. The directory
    # contents are then restored by the script `restore-all-dir-contents`
    # during the container start.
    chown -R 65534:65534 /etc/pki /etc/ssl && save-dir-contents /etc/pki/ca-trust /etc/ssl

# This is equivalent to nobody:nobody.
USER 65534:65534

ENTRYPOINT ["entrypoint.sh"]

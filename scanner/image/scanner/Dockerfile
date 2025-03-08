ARG MAPPINGS_REGISTRY=registry.access.redhat.com
ARG MAPPINGS_BASE_IMAGE=ubi8
ARG MAPPINGS_BASE_TAG=latest
ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8-minimal
ARG BASE_TAG=latest

FROM ${MAPPINGS_REGISTRY}/${MAPPINGS_BASE_IMAGE}:${MAPPINGS_BASE_TAG} AS mappings

COPY download-mappings.sh /download-mappings.sh
RUN /download-mappings.sh /mappings

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

ARG LABEL_VERSION
ARG LABEL_RELEASE
ARG QUAY_TAG_EXPIRATION

LABEL name="scanner-v4" \
      vendor="StackRox" \
      maintainer="https://stackrox.io/" \
      summary="Static vulnerability scanner for the StackRox Security Platform" \
      description="This image supports static vulnerability scanning for the StackRox Security Platform." \
      version="${LABEL_VERSION}" \
      release="${LABEL_RELEASE}" \
      quay.expires-after="${QUAY_TAG_EXPIRATION}"

SHELL ["/bin/sh", "-o", "pipefail", "-c"]

COPY scripts/entrypoint.sh \
     scripts/import-additional-cas \
     scripts/restore-all-dir-contents \
     scripts/save-dir-contents /usr/local/bin/
COPY bin/scanner /usr/local/bin/
COPY THIRD_PARTY_NOTICES/ /THIRD_PARTY_NOTICES/
COPY --from=mappings /mappings/repository-to-cpe.json /mappings/container-name-repos-map.json /run/mappings/

RUN microdnf upgrade --nobest && \
    microdnf clean all && \
    # (Optional) Remove line below to keep package management utilities
    rpm -v -e $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*' libmodulemd) && \
    rm -rf /var/cache/dnf /var/cache/yum && \
    chown -R 65534:65534 /tmp && \
    # The contents of paths mounted as emptyDir volumes in Kubernetes are saved
    # by the script `save-dir-contents` during the image build. The directory
    # contents are then restored by the script `restore-all-dir-contents`
    # during the container start.
    chown -R 65534:65534 /etc/pki/ca-trust /etc/ssl && save-dir-contents /etc/pki/ca-trust /etc/ssl

# This is equivalent to nobody:nobody.
USER 65534:65534

ENTRYPOINT ["entrypoint.sh"]

ARG MAPPINGS_REGISTRY=registry.access.redhat.com
ARG MAPPINGS_BASE_IMAGE=ubi9
ARG MAPPINGS_BASE_TAG=latest
ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi9-minimal
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
     scripts/bundle-ca-trust \
     scripts/restore-all-dir-contents \
     scripts/save-dir-contents /usr/local/bin/
COPY bin/scanner /usr/local/bin/
COPY THIRD_PARTY_NOTICES/ /THIRD_PARTY_NOTICES/
COPY --from=mappings /mappings/repository-to-cpe.json /mappings/container-name-repos-map.json /run/mappings/

RUN microdnf upgrade --nobest && \
    microdnf clean all && \
    # (Optional) Remove line below to keep package management utilities
    rpm -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum && \
    chown -R 65534:65534 /tmp && \
    # Save system CA bundle for bundle-ca-trust (before emptyDir mount hides it)
    mkdir -p /usr/share/pki && \
    cp /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem /usr/share/pki/ca-trust-source.pem

# This is equivalent to nobody:nobody.
USER 65534:65534

ENTRYPOINT ["entrypoint.sh"]

FROM registry.access.redhat.com/ubi9/ubi:latest AS mappings

COPY download-mappings.sh /download-mappings.sh
RUN /download-mappings.sh /mappings

# Build Delve debugger when DEBUG_BUILD=yes.
FROM registry.access.redhat.com/ubi9/ubi:latest AS debugger

ARG DEBUG_BUILD=no

COPY download-dlv.sh /download-dlv.sh
RUN DEBUG_BUILD=${DEBUG_BUILD} /download-dlv.sh

FROM registry.access.redhat.com/ubi9/ubi-micro:latest AS ubi-micro-base

FROM registry.access.redhat.com/ubi9/ubi:latest AS package_installer

COPY --from=ubi-micro-base / /out/

RUN dnf install -y \
    --installroot=/out/ \
    --releasever=9 \
    --setopt=install_weak_deps=0 \
    --nodocs \
    ca-certificates && \
    dnf clean all --installroot=/out/ && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum

FROM ubi-micro-base

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
COPY --from=debugger /output/go/ /go/

COPY --from=package_installer /out/ /

RUN \
    chown -R 65534:65534 /tmp && \
    # The contents of paths mounted as emptyDir volumes in Kubernetes are saved
    # by the script `save-dir-contents` during the image build. The directory
    # contents are then restored by the script `restore-all-dir-contents`
    # during the container start.
    chown -R 65534:65534 /etc/pki/ca-trust && save-dir-contents /etc/pki/ca-trust/source

# This is equivalent to nobody:nobody.
USER 65534:65534

ENTRYPOINT ["entrypoint.sh"]

# The main image: pre-built base + content-addressed artifacts.
# Mirrors the final stage of image/rhel/Dockerfile (the runtime-base FROM,
# package installation, and RPM handling live in the pre-built base image).
ARG TARGET_ARCH=amd64

FROM quay.io/stackrox-io/main:base-${TARGET_ARCH}

ARG TARGET_ARCH
ARG LABEL_VERSION
ARG LABEL_RELEASE
ARG QUAY_TAG_EXPIRATION=1w
ARG ROX_IMAGE_FLAVOR
ARG ROX_PRODUCT_BRANDING

LABEL name="main" \
      vendor="StackRox" \
      maintainer="https://stackrox.io/" \
      summary="The StackRox Kubernetes Security Platform" \
      description="This image contains components required to operate the StackRox Kubernetes Security Platform." \
      version="${LABEL_VERSION}" \
      release="${LABEL_RELEASE}" \
      quay.expires-after="${QUAY_TAG_EXPIRATION}"

ENV PATH="/stackrox:$PATH" \
    ROX_ROXCTL_IN_MAIN_IMAGE="true" \
    ROX_IMAGE_FLAVOR=${ROX_IMAGE_FLAVOR} \
    ROX_PRODUCT_BRANDING=${ROX_PRODUCT_BRANDING}

COPY static-bin /stackrox/

RUN mkdir -p /go/bin && \
    # The contents of paths mounted as emptyDir volumes in Kubernetes are saved
    # by the script `save-dir-contents` during the image build. The directory
    # contents are then restored by the script `restore-all-dir-contents`
    # during the container start.
    chown -R 4000:4000 /etc/pki/ca-trust && save-dir-contents /etc/pki/ca-trust/source && \
    mkdir -p /var/lib/stackrox && chown -R 4000:4000 /var/lib/stackrox && \
    mkdir -p /var/log/stackrox && chown -R 4000:4000 /var/log/stackrox && \
    mkdir -p /var/cache/stackrox && chown -R 4000:4000 /var/cache/stackrox && \
    chown -R 4000:4000 /tmp

COPY static-data/ /stackrox/static-data/
COPY THIRD_PARTY_NOTICES/ /THIRD_PARTY_NOTICES/
COPY ui/ /ui

COPY --link bin/linux_${TARGET_ARCH}/central            /stackrox/central
COPY --link bin/linux_${TARGET_ARCH}/migrator           /stackrox/bin/migrator
COPY --link bin/linux_${TARGET_ARCH}/compliance         /stackrox/bin/compliance
COPY --link bin/linux_${TARGET_ARCH}/kubernetes         /stackrox/bin/kubernetes-sensor
COPY --link bin/linux_${TARGET_ARCH}/admission-control  /stackrox/bin/admission-control
COPY --link bin/linux_${TARGET_ARCH}/config-controller  /stackrox/bin/config-controller
COPY --link bin/linux_${TARGET_ARCH}/worker             /stackrox/bin/central-worker
COPY --link bin/linux_${TARGET_ARCH}/roxagent           /stackrox/bin/roxagent
COPY --link cli/roxctl-*.tar.gz /assets/downloads/cli/
COPY --link cli/roxctl-linux-${TARGET_ARCH} /stackrox/roxctl

EXPOSE 8443

USER 4000:4000

ENTRYPOINT ["/stackrox/roxctl"]

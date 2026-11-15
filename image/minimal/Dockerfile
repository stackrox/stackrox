ARG TARGET_ARCH=amd64
FROM quay.io/stackrox-io/main:base-${TARGET_ARCH}
ARG TARGET_ARCH

COPY --link bin/linux_${TARGET_ARCH}/central /stackrox/central
COPY --link bin/linux_${TARGET_ARCH}/config-controller /stackrox/bin/config-controller
COPY --link bin/linux_${TARGET_ARCH}/worker /stackrox/bin/central-worker
COPY --link ui/build/ /ui/

EXPOSE 8443
USER 4000:4000
ENTRYPOINT ["/stackrox/central"]

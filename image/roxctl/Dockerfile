FROM registry.access.redhat.com/ubi8/ubi-micro:latest AS ubi-micro-base

FROM registry.access.redhat.com/ubi8/ubi:latest AS package_installer

# Copy ubi-micro base to /out/ to preserve its rpmdb
COPY --from=ubi-micro-base / /out/

# Install packages directly to /out/ using --installroot
RUN dnf install -y \
    --installroot=/out/ \
    --releasever=8 \
    --setopt=install_weak_deps=False \
    --nodocs \
    ca-certificates openssl && \
    dnf clean all --installroot=/out/ && \
    rm -rf /out/var/cache/*

COPY ./roxctl-linux /out/usr/bin/roxctl

FROM ubi-micro-base
COPY --from=package_installer /out/ /
ENV ROX_ROXCTL_IN_MAIN_IMAGE="true"

USER 65534:65534
ENTRYPOINT ["/usr/bin/roxctl"]

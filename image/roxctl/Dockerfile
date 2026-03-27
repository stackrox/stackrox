FROM registry.access.redhat.com/ubi9/ubi-micro:latest AS ubi-micro-base

FROM registry.access.redhat.com/ubi9/ubi:latest AS package_installer

# Copy only rpmdb to preserve package state, not entire filesystem
COPY --from=ubi-micro-base /var/lib/rpm /out/var/lib/rpm

# Install packages directly to /out/ using --installroot
RUN dnf install -y \
    --installroot=/out/ \
    --releasever=9 \
    --setopt=install_weak_deps=False \
    --nodocs \
    ca-certificates && \
    dnf clean all --installroot=/out/ && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum

COPY ./roxctl-linux /out/usr/bin/roxctl

FROM ubi-micro-base
COPY --from=package_installer /out/ /
ENV ROX_ROXCTL_IN_MAIN_IMAGE="true"

USER 65534:65534
ENTRYPOINT ["/usr/bin/roxctl"]

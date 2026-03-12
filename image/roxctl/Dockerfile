FROM registry.access.redhat.com/ubi8/ubi-micro:latest AS ubi-micro-base

FROM registry.access.redhat.com/ubi8/ubi:latest AS package_installer

# Copy ubi-micro base to preserve rpmdb
COPY --from=ubi-micro-base / /out/

# Install ca-certificates to /out/ to generate extracted CA bundle
RUN dnf install -y \
        --installroot=/out/ \
        --setopt=reposdir=/etc/yum.repos.d \
        --releasever=8 \
        --setopt=install_weak_deps=0 \
        --nodocs \
        ca-certificates && \
    dnf --installroot=/out/ clean all && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum

FROM scratch
COPY ./roxctl-linux /roxctl
COPY --from=package_installer /out/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem /etc/ssl/certs/ca-certificates.crt
ENV ROX_ROXCTL_IN_MAIN_IMAGE="true"

USER 65534:65534
ENTRYPOINT [ "/roxctl" ]

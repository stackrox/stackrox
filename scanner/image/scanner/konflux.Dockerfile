FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_golang_1.25@sha256:aa03597ee8c7594ffecef5cbb6a0f059d362259d2a41225617b27ec912a3d0d3 AS builder

ARG BUILD_TAG
RUN if [[ "$BUILD_TAG" == "" ]]; then >&2 echo "error: required BUILD_TAG arg is unset"; exit 6; fi
ENV BUILD_TAG="$BUILD_TAG"

ENV GOFLAGS=""
# TODO(ROX-20240): enable non-release development builds.
# TODO(ROX-27054): Remove the redundant strictfipsruntime option if one is found to be so.
ENV GOTAGS="release,strictfipsruntime"
ENV GOEXPERIMENT=strictfipsruntime
ENV CI=1

COPY . /src
WORKDIR /src

RUN make -C scanner NODEPS=1 CGO_ENABLED=1 image/scanner/bin/scanner copy-scripts


FROM registry.access.redhat.com/ubi8/ubi-micro:latest@sha256:37552f11d3b39b3360f7be7c13f6a617e468f39be915cd4f8c8a8531ffc9d43d AS ubi-micro-base

FROM registry.access.redhat.com/ubi8/ubi:latest@sha256:627867e53ad6846afba2dfbf5cef1d54c868a9025633ef0afd546278d4654eac AS package_installer

# Copy ubi-micro base to preserve rpmdb
COPY --from=ubi-micro-base / /out/

# Install packages not in ubi-micro (bash and coreutils-single already present)
RUN dnf install -y \
    --installroot=/out/ \
    --releasever=8 \
    --setopt=install_weak_deps=False \
    --setopt=reposdir=/etc/yum.repos.d \
    --nodocs \
    findutils \
    util-linux \
    ca-certificates && \
    dnf clean all --installroot=/out/ && \
    rm -rf /out/var/cache/*

COPY --from=builder \
    /src/scanner/image/scanner/scripts/entrypoint.sh \
    /src/scanner/image/scanner/scripts/import-additional-cas \
    /src/scanner/image/scanner/scripts/restore-all-dir-contents \
    /src/scanner/image/scanner/scripts/save-dir-contents \
    /src/scanner/image/scanner/bin/scanner \
    /out/usr/local/bin/

# Mapping files required by indexer config
COPY .konflux/scanner-data/repository-to-cpe.json .konflux/scanner-data/container-name-repos-map.json /out/run/mappings/

COPY LICENSE /out/licenses/LICENSE

RUN chown -R 65534:65534 /out/tmp /out/etc/pki/ca-trust /out/etc/ssl && \
    chroot /out /usr/local/bin/save-dir-contents /etc/pki/ca-trust /etc/ssl


FROM ubi-micro-base

COPY --from=package_installer /out/ /

ARG BUILD_TAG

LABEL \
    com.redhat.component="rhacs-scanner-v4-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="This image supports image scanning v4 for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="This image supports image scanning v4 for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="scanner-v4" \
    io.openshift.tags="rhacs,scanner-v4,stackrox" \
    maintainer="Red Hat, Inc." \
    name="advanced-cluster-security/rhacs-scanner-v4-rhel8" \
    # Custom Snapshot creation in `operator-bundle-pipeline` depends on source-location label to be set correctly.
    source-location="https://github.com/stackrox/stackrox" \
    summary="The image scanner v4 for Red Hat Advanced Cluster Security for Kubernetes" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    # We must set version label to prevent inheriting value set in the base stage.
    version="${BUILD_TAG}" \
    # Release label is required by EC although has no practical semantics.
    # We also set it to not inherit one from a base stage in case it's RHEL or UBI.
    release="1"

# This is equivalent to nobody:nobody.
USER 65534:65534

ENTRYPOINT ["entrypoint.sh"]

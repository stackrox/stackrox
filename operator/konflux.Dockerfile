FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_golang_1.25@sha256:aa03597ee8c7594ffecef5cbb6a0f059d362259d2a41225617b27ec912a3d0d3 AS builder

WORKDIR /go/src/github.com/stackrox/rox/app

COPY . .

ARG BUILD_TAG
RUN if [[ "$BUILD_TAG" == "" ]]; then >&2 echo "error: required BUILD_TAG arg is unset"; exit 6; fi
ENV BUILD_TAG="$BUILD_TAG"

# TODO(ROX-20240): enable non-release development builds.
# TODO(ROX-27054): Remove the redundant strictfipsruntime option if one is found to be so.
ENV GOTAGS="release,strictfipsruntime"
ENV GOEXPERIMENT=strictfipsruntime
ENV CI=1 GOFLAGS="" CGO_ENABLED=1

RUN GOOS=linux GOARCH=$(go env GOARCH) scripts/go-build-file.sh operator/cmd/main.go image/bin/operator


FROM registry.access.redhat.com/ubi8/ubi-micro:latest@sha256:37552f11d3b39b3360f7be7c13f6a617e468f39be915cd4f8c8a8531ffc9d43d AS ubi-micro-base


FROM registry.access.redhat.com/ubi8/ubi:latest@sha256:627867e53ad6846afba2dfbf5cef1d54c868a9025633ef0afd546278d4654eac AS package_installer

# Copy ubi-micro base to /out/ to preserve its rpmdb
COPY --from=ubi-micro-base / /out/

# Install packages directly to /out/ using --installroot
# Note: --setopt=reposdir=/etc/yum.repos.d instructs dnf to use repo configurations pointing to RPMs
# prefetched by Hermeto/Cachi2, instead of installroot's default UBI repos.
RUN dnf install -y \
    --installroot=/out/ \
    --releasever=8 \
    --setopt=install_weak_deps=False \
    --setopt=reposdir=/etc/yum.repos.d \
    --nodocs \
    ca-certificates \
    openssl && \
    dnf clean all --installroot=/out/ && \
    rm -rf /out/var/cache/*


FROM ubi-micro-base

ARG BUILD_TAG

LABEL \
    com.redhat.component="rhacs-operator-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="Operator for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="Operator for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="operator" \
    io.openshift.tags="rhacs,operator,stackrox" \
    maintainer="Red Hat, Inc." \
    name="advanced-cluster-security/rhacs-rhel8-operator" \
    # Custom Snapshot creation in `operator-bundle-pipeline` depends on source-location label to be set correctly.
    source-location="https://github.com/stackrox/stackrox" \
    summary="Operator for Red Hat Advanced Cluster Security for Kubernetes" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    # We must set version label to prevent inheriting value set in the base stage.
    version="${BUILD_TAG}" \
    # Release label is required by EC although has no practical semantics.
    # We also set it to not inherit one from a base stage in case it's RHEL or UBI.
    release="1"

COPY --from=package_installer /out/ /

COPY --from=builder /go/src/github.com/stackrox/rox/app/image/bin/operator /usr/local/bin/rhacs-operator

COPY LICENSE /licenses/LICENSE

ENV ROX_IMAGE_FLAVOR="rhacs"

# The following are numeric uid and gid of `nobody` user in UBI.
# We can't use symbolic names because otherwise k8s will fail to start the pod with an error like this:
# Error: container has runAsNonRoot and image has non-numeric user (nobody), cannot verify user is non-root.
USER 65534:65534

ENTRYPOINT ["/usr/local/bin/rhacs-operator"]

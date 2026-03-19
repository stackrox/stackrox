FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.25@sha256:bd531796aacb86e4f97443797262680fbf36ca048717c00b6f4248465e1a7c0c AS builder

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

RUN GOOS=linux GOARCH=$(go env GOARCH) scripts/go-build-file.sh operator/stackrox-operator/main.go image/bin/operator


FROM registry.access.redhat.com/ubi9/ubi-micro:latest@sha256:2173487b3b72b1a7b11edc908e9bbf1726f9df46a4f78fd6d19a2bab0a701f38 AS ubi-micro-base


FROM registry.access.redhat.com/ubi9/ubi:latest@sha256:1fc04e873cb3f3c8d1211729a794716e50826f650ea88b97a4ff57f601db77a8 AS package_installer

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
    name="advanced-cluster-security/rhacs-rhel9-operator" \
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

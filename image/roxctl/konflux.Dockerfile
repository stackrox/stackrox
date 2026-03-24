# openshift-golang-builder is the only way to get more recent Go version than the official ubi8/go-toolset provides.
# See https://issues.redhat.com/browse/RHELPLAN-167618
# Using that has few known issues:
# - https://issues.redhat.com/browse/RHTAPBUGS-864 - deprecated-base-image-check behaves incorrectly.
# - https://issues.redhat.com/browse/RHTAPBUGS-865 - openshift-golang-builder is not considered to be a valid base image.
#
FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.25@sha256:bd531796aacb86e4f97443797262680fbf36ca048717c00b6f4248465e1a7c0c AS builder

WORKDIR /go/src/github.com/stackrox/rox/app

COPY . .

RUN mkdir -p image/bin

ARG BUILD_TAG
RUN if [[ "$BUILD_TAG" == "" ]]; then >&2 echo "error: required BUILD_TAG arg is unset"; exit 6; fi
ENV BUILD_TAG="$BUILD_TAG"

ENV CI=1 GOFLAGS=""
# TODO(ROX-20240): enable non-release development builds.
# TODO(ROX-27054): Remove the redundant strictfipsruntime option if one is found to be so.
ENV GOTAGS="release,strictfipsruntime"
ENV GOEXPERIMENT=strictfipsruntime

RUN RACE=0 CGO_ENABLED=1 GOOS=linux GOARCH=$(go env GOARCH) scripts/go-build.sh ./roxctl && \
    cp bin/linux_$(go env GOARCH)/roxctl image/bin/roxctl

FROM registry.access.redhat.com/ubi9/ubi-micro:latest@sha256:093a704be0eaef9bb52d9bc0219c67ee9db13c2e797da400ddb5d5ae6849fa10 AS ubi-micro-base

FROM registry.access.redhat.com/ubi9/ubi:latest@sha256:6ed9f6f637fe731d93ec60c065dbced79273f1e0b5f512951f2c0b0baedb16ad AS package_installer

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
    ca-certificates openssl && \
    dnf clean all --installroot=/out/ && \
    rm -rf /out/var/cache/*

COPY --from=builder /go/src/github.com/stackrox/rox/app/image/bin/roxctl /out/usr/bin/roxctl
COPY LICENSE /out/licenses/LICENSE

FROM ubi-micro-base

COPY --from=package_installer /out/ /

ARG BUILD_TAG

LABEL \
    com.redhat.component="rhacs-roxctl-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="The CLI for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="The CLI for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="roxctl" \
    io.openshift.tags="rhacs,roxctl,stackrox" \
    maintainer="Red Hat, Inc." \
    name="advanced-cluster-security/rhacs-roxctl-rhel9" \
    # Custom Snapshot creation in `operator-bundle-pipeline` depends on source-location label to be set correctly.
    source-location="https://github.com/stackrox/stackrox" \
    summary="The CLI for Red Hat Advanced Cluster Security for Kubernetes" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    # We must set version label to prevent inheriting value set in the base stage.
    version="${BUILD_TAG}" \
    # Release label is required by EC although has no practical semantics.
    # We also set it to not inherit one from a base stage in case it's RHEL or UBI.
    release="1"

ENV ROX_ROXCTL_IN_MAIN_IMAGE="true"

USER 65534:65534

ENTRYPOINT ["/usr/bin/roxctl"]

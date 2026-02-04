# openshift-golang-builder is the only way to get more recent Go version than the official ubi8/go-toolset provides.
# See https://issues.redhat.com/browse/RHELPLAN-167618
# Using that has few known issues:
# - https://issues.redhat.com/browse/RHTAPBUGS-864 - deprecated-base-image-check behaves incorrectly.
# - https://issues.redhat.com/browse/RHTAPBUGS-865 - openshift-golang-builder is not considered to be a valid base image.
#
FROM registry.redhat.io/rhel8/go-toolset:1.25@sha256:070a8b01fe3c47cda74390c49c37e0abde1157c162cc3a46be1698564a18f923 AS builder

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


FROM registry.access.redhat.com/ubi8/ubi-minimal:latest@sha256:5dc6ba426ccbeb3954ead6b015f36b4a2d22320e5b356b074198d08422464ed2

COPY --from=builder /go/src/github.com/stackrox/rox/app/image/bin/roxctl /usr/bin/roxctl

RUN microdnf clean all && \
    rpm --verbose -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum

COPY LICENSE /licenses/LICENSE

ARG BUILD_TAG

LABEL \
    com.redhat.component="rhacs-roxctl-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="The CLI for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="The CLI for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="roxctl" \
    io.openshift.tags="rhacs,roxctl,stackrox" \
    maintainer="Red Hat, Inc." \
    name="advanced-cluster-security/rhacs-roxctl-rhel8" \
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

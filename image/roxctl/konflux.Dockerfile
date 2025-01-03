# openshift-golang-builder is the only way to get more recent Go version than the official ubi8/go-toolset provides.
# See https://issues.redhat.com/browse/RHELPLAN-167618
# Using that has few known issues:
# - https://issues.redhat.com/browse/RHTAPBUGS-864 - deprecated-base-image-check behaves incorrectly.
# - https://issues.redhat.com/browse/RHTAPBUGS-865 - openshift-golang-builder is not considered to be a valid base image.
#
# TODO(ROX-20312): we can't pin image tag or digest because currently there's no mechanism to auto-update that.
# We're targeting a floating tag here which should be reasonably safe to do as both RHEL major 8 and Go major.minor 1.22 should provide enough stability.
FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_1.22 AS builder

WORKDIR /go/src/github.com/stackrox/rox/app

COPY . .

RUN .konflux/scripts/fail-build-if-git-is-dirty.sh

RUN mkdir -p image/bin

ARG VERSIONS_SUFFIX
ENV MAIN_TAG_SUFFIX="$VERSIONS_SUFFIX" COLLECTOR_TAG_SUFFIX="$VERSIONS_SUFFIX" SCANNER_TAG_SUFFIX="$VERSIONS_SUFFIX"

ENV CI=1 GOFLAGS=""
# TODO(ROX-20240): enable non-release development builds.
ENV GOTAGS="release"

RUN RACE=0 CGO_ENABLED=1 GOOS=linux GOARCH=$(go env GOARCH) scripts/go-build.sh ./roxctl && \
    cp bin/linux_$(go env GOARCH)/roxctl image/bin/roxctl


# TODO(ROX-20312): pin image tags when there's a process that updates them automatically.
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

COPY --from=builder /go/src/github.com/stackrox/rox/app/image/bin/roxctl /usr/bin/roxctl

# TODO(ROX-20234): use hermetic builds when installing/updating RPMs becomes hermetic.
RUN microdnf upgrade -y --nobest && \
    microdnf clean all && \
    rpm --verbose -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum

COPY LICENSE /licenses/LICENSE

ARG MAIN_IMAGE_TAG
RUN if [[ "$MAIN_IMAGE_TAG" == "" ]]; then >&2 echo "error: required MAIN_IMAGE_TAG arg is unset"; exit 6; fi

LABEL \
    com.redhat.component="rhacs-roxctl-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="The CLI for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="The CLI for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="roxctl" \
    io.openshift.tags="rhacs,roxctl,stackrox" \
    maintainer="Red Hat, Inc." \
    name="rhacs-roxctl-rhel8" \
    # Custom Snapshot creation in `operator-bundle-pipeline` depends on source-location label to be set correctly.
    source-location="https://github.com/stackrox/stackrox" \
    summary="The CLI for Red Hat Advanced Cluster Security for Kubernetes" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    # We must set version label to prevent inheriting value set in the base stage.
    version="${MAIN_IMAGE_TAG}" \
    # Release label is required by EC although has no practical semantics.
    # We also set it to not inherit one from a base stage in case it's RHEL or UBI.
    release="1"

ENV ROX_ROXCTL_IN_MAIN_IMAGE="true"

USER 65534:65534

ENTRYPOINT ["/usr/bin/roxctl"]

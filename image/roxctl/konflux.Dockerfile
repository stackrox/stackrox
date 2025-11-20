FROM registry.access.redhat.com/ubi9/go-toolset:1.24@sha256:c06c8764041cceae3ef35962b44043bfea534ad336d2cb14cafb5d0c384a5b5e AS builder

# Build rootless using $APP_ROOT which is accessible to the default user
# $APP_ROOT is typically /opt/app-root/ in UBI images
ENV GOPATH=${APP_ROOT:-/opt/app-root/}/src
WORKDIR ${GOPATH}/github.com/stackrox/rox/app

# Copy files with ownership set to default user so we have permissions to create directories
COPY --chown=default . .

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


FROM registry.access.redhat.com/ubi9/ubi-minimal:latest@sha256:2ddd6e10383981c7d10e4966a7c0edce7159f8ca91b1691cafabc78bae79d8f8

COPY --from=builder /opt/app-root/src/github.com/stackrox/rox/app/image/bin/roxctl /usr/bin/roxctl

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

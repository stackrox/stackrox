# openshift-golang-builder is the only way to get more recent Go version than the official ubi8/go-toolset provides.
# See https://issues.redhat.com/browse/RHELPLAN-167618
# Using that has few known issues:
# - https://issues.redhat.com/browse/RHTAPBUGS-864 - deprecated-base-image-check behaves incorrectly.
# - https://issues.redhat.com/browse/RHTAPBUGS-865 - openshift-golang-builder is not considered to be a valid base image.
#
# Also, we can't pin image tag or digest because currently there's no mechanism to auto-update that.
# See https://issues.redhat.com/browse/STONEBLD-1823
# We're targeting a floating tag here which should be reasonably safe to do as both RHEL major 8 and Go major.minor 1.20 should provide enough stability.
FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_1.20 as builder

WORKDIR /go/src/github.com/stackrox/rox/app

COPY . .

RUN git config --global --add safe.directory /go/src/github.com/stackrox/rox/app && \
    # TODO: this prevents hermetic builds
    # See related: https://redhat-internal.slack.com/archives/C04PZ7H0VA8/p1696941513601719 and https://github.com/redhat-appstudio/build-definitions/pull/615
    git fetch --tags --force && \
    mkdir -p image/bin

ENV CI=1 GOFLAGS="" GOTAGS="release"

RUN RACE=0 CGO_ENABLED=1 GOOS=linux GOARCH=$(go env GOARCH) BUILD_TAG=$(make tag) scripts/go-build.sh ./roxctl && \
    cp bin/linux_$(go env GOARCH)/roxctl image/bin/roxctl

# TODO: pin image tags when there's a process that updates them automatically.
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

COPY --from=builder /go/src/github.com/stackrox/rox/app/image/bin/roxctl /usr/bin/roxctl

# TODO: use hermetic builds when installing/updating RPMs becomes hermetic.
# See https://issues.redhat.com/browse/STONEBLD-704
RUN microdnf upgrade -y --nobest && \
    microdnf clean all && \
    rpm --verbose -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum

LABEL \
    com.redhat.component="rhacs-roxctl-container" \
    name="rhacs-roxctl-rhel8" \
    maintainer="Red Hat, Inc." \
    source-location="https://github.com/stackrox/stackrox" \
    # These labels are added to override the base image values.
    description="The CLI for ACS" \
    io.k8s.description="The CLI for ACS" \
    io.k8s.display-name="roxctl" \
    io.openshift.tags="rhacs,roxctl,stackrox" \
    summary="The CLI for ACS" \
    # If we don't reset the following labels, we inherit values from the base container which will be incorrect.
    # At the same time, we can't configure correct values yet. E.g. see the following thread about version:
    # https://redhat-internal.slack.com/archives/C04PZ7H0VA8/p1697127151309229
    com.redhat.license_terms="" \
    url="" \
    version=""

ENV ROX_ROXCTL_IN_MAIN_IMAGE="true"

USER 65534:65534

ENTRYPOINT ["/usr/bin/roxctl"]

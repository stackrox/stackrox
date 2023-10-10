# TODO: can we follow tags?
FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_1.20 as builder

WORKDIR /go/src/github.com/stackrox/rox/app

# TODO: do we want to avoid this by COPYing each required directory separately?
COPY . .

RUN git config --global --add safe.directory /go/src/github.com/stackrox/rox/app && \
    make tag && \
    mkdir -p image/bin

ENV CI=1 GOFLAGS="" GOTAGS="release"

RUN RACE=0 CGO_ENABLED=1 GOOS=linux GOARCH=$(go env GOARCH) BUILD_TAG=$(make tag) scripts/go-build.sh ./roxctl && \
    cp bin/linux_$(go env GOARCH)/roxctl image/bin/roxctl

# TODO: can we follow tags?
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest as app

COPY --from=builder /go/src/github.com/stackrox/rox/app/image/bin/roxctl /usr/bin/roxctl

# TODO: this prevents hermetic builds
RUN microdnf upgrade -y --nobest && \
    microdnf clean all && \
    rpm --verbose -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum

LABEL \
    com.redhat.component="rhacs-roxctl-container" \
    name="rhacs-roxctl-rhel8" \
    maintainer="Red Hat, Inc." \
    # These labels are added to override the base image values.
    description="The CLI for ACS" \
    io.k8s.description="The CLI for ACS" \
    io.k8s.display-name="roxctl" \
    io.openshift.tags="rhacs,roxctl" \
    summary="The CLI for ACS"

    # TODO: what are these for & if they're required, how can we ?
    # version="${CI_VERSION}" \
    # "git-commit:stackrox/stackrox"="${CI_STACKROX_UPSTREAM_COMMIT}" \ --> vcs-ref on RHTAP
    # "git-branch:stackrox/stackrox"="${CI_STACKROX_UPSTREAM_BRANCH}" \
    # "git-tag:stackrox/stackrox"="${CI_STACKROX_UPSTREAM_TAG}"

ENV ROX_ROXCTL_IN_MAIN_IMAGE="true"

USER 65534:65534

ENTRYPOINT ["/usr/bin/roxctl"]

# TODO(ROX-20312): we can't pin image tag or digest because currently there's no mechanism to auto-update that.
# We're targeting a floating tag here which should be reasonably safe to do as both RHEL major 8 and Go major.minor 1.20 should provide enough stability.
FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_1.20 as builder

WORKDIR /go/src/github.com/stackrox/rox/app

COPY . .

RUN git status && scripts/rhtap/fail-build-if-git-is-dirty.sh

RUN mkdir -p image/bin

# Build the operator binary.
# TODO(ROX-20240): enable non-release development builds.
ENV CI=1 GOFLAGS="" GOTAGS="release"

RUN RACE=0 CGO_ENABLED=1 GOOS=linux GOARCH=$(go env GOARCH) BUILD_TAG=$(make tag) scripts/go-build.sh ./operator && \
    cp bin/linux_$(go env GOARCH)/operator image/bin/operator

# TODO(ROX-20312): pin image tags when there's a process that updates them automatically.
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

LABEL \
    com.redhat.component="rhacs-operator-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="TODO" \
    io.k8s.description="TODO" \
    io.k8s.display-name="operator" \
    io.openshift.tags="rhacs,operator,stackrox" \
    maintainer="Red Hat, Inc." \
    name="rhacs-operator-rhel8" \
    source-location="https://github.com/stackrox/stackrox" \
    summary="TODO" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    # We must set version label to prevent inheriting value set in the base stage.
    # TODO(ROX-20236): configure injection of dynamic version value when it becomes possible.
    version="0.0.1-todo"

COPY --from=builder /go/src/github.com/stackrox/rox/app/image/bin/operator /usr/bin/operator

# TODO(ROX-20234): use hermetic builds when installing/updating RPMs becomes hermetic.
RUN microdnf upgrade -y --nobest && \
    microdnf clean all && \
    rpm --verbose -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum

ENV ROX_IMAGE_FLAVOR="rhacs"

USER 65534:65534

ENTRYPOINT ["/usr/bin/rhacs-operator"]

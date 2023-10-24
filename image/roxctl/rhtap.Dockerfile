# openshift-golang-builder is the only way to get more recent Go version than the official ubi8/go-toolset provides.
# See https://issues.redhat.com/browse/RHELPLAN-167618
# Using that has few known issues:
# - https://issues.redhat.com/browse/RHTAPBUGS-864 - deprecated-base-image-check behaves incorrectly.
# - https://issues.redhat.com/browse/RHTAPBUGS-865 - openshift-golang-builder is not considered to be a valid base image.
#
# TODO(ROX-20312): we can't pin image tag or digest because currently there's no mechanism to auto-update that.
# We're targeting a floating tag here which should be reasonably safe to do as both RHEL major 8 and Go major.minor 1.20 should provide enough stability.
FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_1.20 as builder

WORKDIR /go/src/github.com/stackrox/rox/app

COPY . .

RUN git config --global --add safe.directory /go/src/github.com/stackrox/rox/app && \
    # TODO(ROX-20233): Fetch git tags outside of Dockerfile
    git tag
#     # git fetch --tags --force && \
#     mkdir -p image/bin

# # TODO(ROX-20240): enable non-release development builds.
# ENV CI=1 GOFLAGS="" GOTAGS="release"

# RUN RACE=0 CGO_ENABLED=1 GOOS=linux GOARCH=$(go env GOARCH) BUILD_TAG=$(make tag) scripts/go-build.sh ./roxctl && \
#     cp bin/linux_$(go env GOARCH)/roxctl image/bin/roxctl

# # TODO(ROX-20312): pin image tags when there's a process that updates them automatically.
# FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

# COPY --from=builder /go/src/github.com/stackrox/rox/app/image/bin/roxctl /usr/bin/roxctl

# # TODO(ROX-20234): use hermetic builds when installing/updating RPMs becomes hermetic.
# RUN microdnf upgrade -y --nobest && \
#     microdnf clean all && \
#     rpm --verbose -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
#     rm -rf /var/cache/dnf /var/cache/yum

# LABEL \
#     com.redhat.component="rhacs-roxctl-container" \
#     com.redhat.license_terms="https://www.redhat.com/agreements" \
#     description="The CLI for RHACS" \
#     io.k8s.description="The CLI for RHACS" \
#     io.k8s.display-name="roxctl" \
#     io.openshift.tags="rhacs,roxctl,stackrox" \
#     maintainer="Red Hat, Inc." \
#     name="rhacs-roxctl-rhel8" \
#     source-location="https://github.com/stackrox/stackrox" \
#     summary="The CLI for RHACS" \
#     url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
#     # We must set version label to prevent inheriting value set in the base stage.
#     # TODO(ROX-20236): configure injection of dynamic version value when it becomes possible.
#     version="0.0.1-todo"

# ENV ROX_ROXCTL_IN_MAIN_IMAGE="true"

# USER 65534:65534

# ENTRYPOINT ["/usr/bin/roxctl"]

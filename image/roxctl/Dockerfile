FROM registry.access.redhat.com/ubi8/go-toolset:1.19 as builder

WORKDIR /go/src/github.com/stackrox/rox/app

RUN git config --global --add safe.directory /go/src/github.com/stackrox/rox/app && \
    mkdir -p image/bin

COPY . .

ENV CI=1 BUILD_TAG="4.2.x-275-g1e681e191b" GOFLAGS="" GOTAGS="release"

RUN RACE=0 CGO_ENABLED=1 GOOS=linux GOARCH=$(go env GOARCH) scripts/go-build.sh ./roxctl && \
    cp bin/linux_$(go env GOARCH)/roxctl image/bin/roxctl

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest as app

COPY --from=builder /go/src/github.com/stackrox/rox/app/image/bin/roxctl /usr/bin/roxctl

RUN microdnf upgrade -y --nobest && \
    microdnf clean all && \
    rpm --verbose -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum

LABEL \
    com.redhat.component="rhacs-roxctl-container" \
    name="rhacs-roxctl-rhel8" \
    maintainer="Red Hat, Inc."
    # version="${CI_VERSION}" \
    # "git-commit:stackrox/stackrox"="${CI_STACKROX_UPSTREAM_COMMIT}" \
    # "git-branch:stackrox/stackrox"="${CI_STACKROX_UPSTREAM_BRANCH}" \
    # "git-tag:stackrox/stackrox"="${CI_STACKROX_UPSTREAM_TAG}"

ENV ROX_ROXCTL_IN_MAIN_IMAGE="true"

USER 65534:65534

ENTRYPOINT ["/usr/bin/roxctl"]

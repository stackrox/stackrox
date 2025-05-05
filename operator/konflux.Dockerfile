# TODO(ROX-20312): we can't pin image tag or digest because currently there's no mechanism to auto-update that.
# We're targeting a floating tag here which should be reasonably safe to do as both RHEL major 8 and Go major.minor 1.20 should provide enough stability.
FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_1.23 AS builder

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

RUN GOOS=linux GOARCH=$(go env GOARCH) scripts/go-build-file.sh operator/cmd/main.go image/bin/operator


# TODO(ROX-20312): pin image tags when there's a process that updates them automatically.
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ARG BUILD_TAG

LABEL \
    com.redhat.component="rhacs-operator-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="Operator for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="Operator for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="operator" \
    io.openshift.tags="rhacs,operator,stackrox" \
    maintainer="Red Hat, Inc." \
    name="rhacs-rhel8-operator" \
    # Custom Snapshot creation in `operator-bundle-pipeline` depends on source-location label to be set correctly.
    source-location="https://github.com/stackrox/stackrox" \
    summary="Operator for Red Hat Advanced Cluster Security for Kubernetes" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    # We must set version label to prevent inheriting value set in the base stage.
    version="${BUILD_TAG}" \
    # Release label is required by EC although has no practical semantics.
    # We also set it to not inherit one from a base stage in case it's RHEL or UBI.
    release="1"

COPY --from=builder /go/src/github.com/stackrox/rox/app/image/bin/operator /usr/local/bin/rhacs-operator

RUN microdnf clean all && \
    rpm --verbose -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum

COPY LICENSE /licenses/LICENSE

ENV ROX_IMAGE_FLAVOR="rhacs"

# The following are numeric uid and gid of `nobody` user in UBI.
# We can't use symbolic names because otherwise k8s will fail to start the pod with an error like this:
# Error: container has runAsNonRoot and image has non-numeric user (nobody), cannot verify user is non-root.
USER 65534:65534

ENTRYPOINT ["/usr/local/bin/rhacs-operator"]

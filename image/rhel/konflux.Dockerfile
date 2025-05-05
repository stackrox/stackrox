ARG PG_VERSION=15


# TODO(ROX-20312): we can't pin image tag or digest because currently there's no mechanism to auto-update that.
FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_1.23 AS go-builder

RUN dnf -y install --allowerasing jq

WORKDIR /go/src/github.com/stackrox/rox/app

COPY . .

ARG BUILD_TAG
RUN if [[ "$BUILD_TAG" == "" ]]; then >&2 echo "error: required BUILD_TAG arg is unset"; exit 6; fi
ENV BUILD_TAG="$BUILD_TAG"

ENV GOFLAGS=""
ENV CGO_ENABLED=1
# TODO(ROX-20240): enable non-release development builds.
# TODO(ROX-27054): Remove the redundant strictfipsruntime option if one is found to be so.
ENV GOTAGS="release,strictfipsruntime"
ENV GOEXPERIMENT=strictfipsruntime
ENV CI=1

RUN # TODO(ROX-13200): make sure roxctl cli is built without running go mod tidy. \
    make main-build-nodeps cli-build

RUN mkdir -p image/rhel/docs/api/v1 && \
    ./scripts/mergeswag.sh 1 generated/api/v1 central/docs/api_custom_routes >image/rhel/docs/api/v1/swagger.json && \
    mkdir -p image/rhel/docs/api/v2 && \
    ./scripts/mergeswag.sh 2 generated/api/v2 >image/rhel/docs/api/v2/swagger.json

RUN make copy-go-binaries-to-image-dir


FROM registry.access.redhat.com/ubi8/nodejs-20:latest AS ui-builder

WORKDIR /go/src/github.com/stackrox/rox/app

COPY --chown=default . .

# This sets branding during UI build time. This is to make sure UI is branded as commercial RHACS (not StackRox).
# ROX_PRODUCT_BRANDING is also set in the resulting image so that Central Go code knows its RHACS.
ENV ROX_PRODUCT_BRANDING="RHACS_BRANDING"

# Default execution of the `npm ci` command causes postinstall scripts to run and spawn a new child process
# for each script. When building in konflux for s390x and ppc64le architectures, spawing
# these child processes causes excessive memory usage and ENOMEM errors, resulting
# in build failures. Currently the only postinstall scripts that run for the UI dependencies are:
#   `core-js` prints a banner with links for donations
#   `cypress` downloads the Cypress binary from the internet
# In the case of building the `rhacs-main-container`, all of these install scripts can be safely ignored.
ENV UI_PKG_INSTALL_EXTRA_ARGS="--ignore-scripts"

RUN make -C ui build


FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ARG PG_VERSION

RUN microdnf -y module enable postgresql:${PG_VERSION} && \
    # find is used in /stackrox/import-additional-cas \
    microdnf -y install findutils postgresql && \
    microdnf -y clean all && \
    rpm --verbose -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum

COPY --from=ui-builder /go/src/github.com/stackrox/rox/app/ui/build /ui/

COPY --from=go-builder /go/src/github.com/stackrox/rox/app/image/rhel/bin/migrator /stackrox/bin/
COPY --from=go-builder /go/src/github.com/stackrox/rox/app/image/rhel/bin/central /stackrox/
COPY --from=go-builder /go/src/github.com/stackrox/rox/app/image/rhel/bin/compliance /stackrox/bin/
COPY --from=go-builder /go/src/github.com/stackrox/rox/app/image/rhel/bin/roxctl* /assets/downloads/cli/
COPY --from=go-builder /go/src/github.com/stackrox/rox/app/image/rhel/bin/kubernetes-sensor /stackrox/bin/
COPY --from=go-builder /go/src/github.com/stackrox/rox/app/image/rhel/bin/sensor-upgrader /stackrox/bin/
COPY --from=go-builder /go/src/github.com/stackrox/rox/app/image/rhel/bin/admission-control /stackrox/bin/
COPY --from=go-builder /go/src/github.com/stackrox/rox/app/image/rhel/bin/config-controller /stackrox/bin/
COPY --from=go-builder /go/src/github.com/stackrox/rox/app/image/rhel/bin/init-tls-certs /stackrox/bin/
COPY --from=go-builder /go/src/github.com/stackrox/rox/app/image/rhel/static-bin/* /stackrox/
RUN GOARCH=$(uname -m) ; \
    case $GOARCH in x86_64) GOARCH=amd64 ;; aarch64) GOARCH=arm64 ;; esac ; \
    ln -s /assets/downloads/cli/roxctl-linux-$GOARCH /stackrox/roxctl ; \
    ln -s /assets/downloads/cli/roxctl-linux-$GOARCH /assets/downloads/cli/roxctl-linux

ARG BUILD_TAG

LABEL \
    com.redhat.component="rhacs-main-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="Main Image for Red Hat Advanced Cluster Security for Kubernetes" \
    distribution-scope="public" \
    io.k8s.description="Main Image for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="main" \
    io.openshift.tags="rhacs,main,stackrox" \
    maintainer="Red Hat, Inc." \
    name="rhacs-main-rhel8" \
    # Custom Snapshot creation in `operator-bundle-pipeline` depends on source-location label to be set correctly.
    source-location="https://github.com/stackrox/stackrox" \
    summary="Main Image for Red Hat Advanced Cluster Security for Kubernetes" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    vendor="Red Hat, Inc." \
    # We must set version label to prevent inheriting value set in the base stage.
    version="${BUILD_TAG}" \
    # Release label is required by EC although has no practical semantics.
    # We also set it to not inherit one from a base stage in case it's RHEL or UBI.
    release="1"

EXPOSE 8443

ENV PATH="/stackrox:$PATH" \
    ROX_ROXCTL_IN_MAIN_IMAGE="true" \
    ROX_IMAGE_FLAVOR="rhacs" \
    ROX_PRODUCT_BRANDING="RHACS_BRANDING"

COPY .konflux/stackrox-data/external-networks/external-networks.zip /stackrox/static-data/external-networks/external-networks.zip

COPY --from=go-builder /go/src/github.com/stackrox/rox/app/image/rhel/docs/api/v1/swagger.json /stackrox/static-data/docs/api/v1/swagger.json
COPY --from=go-builder /go/src/github.com/stackrox/rox/app/image/rhel/docs/api/v2/swagger.json /stackrox/static-data/docs/api/v2/swagger.json

COPY LICENSE /licenses/LICENSE

# The following paths are written to in Central.
RUN chown -R 4000:4000 /etc/pki/ca-trust /etc/ssl && save-dir-contents /etc/pki/ca-trust /etc/ssl && \
    mkdir -p /var/lib/stackrox && chown -R 4000:4000 /var/lib/stackrox && \
    mkdir -p /var/log/stackrox && chown -R 4000:4000 /var/log/stackrox && \
    mkdir -p /var/cache/stackrox && chown -R 4000:4000 /var/cache/stackrox && \
    chown -R 4000:4000 /tmp

USER 4000:4000

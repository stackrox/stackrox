ARG FINAL_STAGE_PATH="/mnt/final"

# TODO(ROX-20312): we can't pin image tag or digest because currently there's no mechanism to auto-update that.
FROM registry.access.redhat.com/ubi8/ubi:latest AS ubi-base
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest AS final-base


# TODO(ROX-20651): use content sets instead of subscription manager for access to RHEL RPMs once available. Move dnf commands to respective stages.
FROM ubi-base AS rpm-installer

ARG FINAL_STAGE_PATH
COPY --from=final-base / "$FINAL_STAGE_PATH"

COPY ./.konflux/scripts/subscription-manager/* /tmp/.konflux/
RUN /tmp/.konflux/subscription-manager-bro.sh register "$FINAL_STAGE_PATH"

# Install packages for the final stage.
RUN dnf -y --installroot="$FINAL_STAGE_PATH" upgrade --nobest && \
    dnf -y --installroot="$FINAL_STAGE_PATH" module enable postgresql:13 && \
    # find is used in /stackrox/import-additional-cas \
    dnf -y --installroot="$FINAL_STAGE_PATH" install findutils postgresql && \
    # We can do usual cleanup while we're here: remove packages that would trigger violations. \
    dnf -y --installroot="$FINAL_STAGE_PATH" clean all && \
    rpm --root="$FINAL_STAGE_PATH" --verbose -e --nodeps $(rpm --root="$FINAL_STAGE_PATH" -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf "$FINAL_STAGE_PATH/var/cache/dnf" "$FINAL_STAGE_PATH/var/cache/yum"

RUN /tmp/.konflux/subscription-manager-bro.sh cleanup


FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_1.23 AS go-builder

RUN dnf -y install --allowerasing make automake gcc gcc-c++ coreutils binutils diffutils zlib-devel bzip2-devel lz4-devel cmake jq

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
    make main-build-nodeps cli-build && \
    mkdir -p image/rhel/docs/api/v1 && \
    ./scripts/mergeswag.sh generated/api/v1 1 >image/rhel/docs/api/v1/swagger.json && \
    mkdir -p image/rhel/docs/api/v2 && \
    ./scripts/mergeswag.sh generated/api/v2 2 >image/rhel/docs/api/v2/swagger.json

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


FROM scratch

ARG FINAL_STAGE_PATH
COPY --from=rpm-installer "$FINAL_STAGE_PATH" /

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

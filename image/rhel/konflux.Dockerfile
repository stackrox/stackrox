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


FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_1.21 AS go-builder

RUN dnf -y install --allowerasing make automake gcc gcc-c++ coreutils binutils diffutils zlib-devel bzip2-devel lz4-devel cmake jq

WORKDIR /go/src/github.com/stackrox/rox/app

COPY . .

# Ensure there will be no unintended -dirty suffix. package-lock is restored because it's touched by Cachi2.
RUN git restore .konflux/bootstrap-yarn/package-lock.json && \
    .konflux/scripts/fail-build-if-git-is-dirty.sh

ARG VERSIONS_SUFFIX
ENV MAIN_TAG_SUFFIX="$VERSIONS_SUFFIX" COLLECTOR_TAG_SUFFIX="$VERSIONS_SUFFIX" SCANNER_TAG_SUFFIX="$VERSIONS_SUFFIX"

ENV GOFLAGS=""
ENV CGO_ENABLED=1
# TODO(ROX-24276): re-enable release builds for fast stream.
# TODO(ROX-20240): enable non-release development builds.
# ENV GOTAGS="release"
ENV CI=1

RUN # TODO(ROX-13200): make sure roxctl cli is built without running go mod tidy. \
    make main-build-nodeps cli-build && \
    mkdir -p image/rhel/docs/api/v1 && \
    ./scripts/mergeswag.sh generated/api/v1 1 >image/rhel/docs/api/v1/swagger.json && \
    mkdir -p image/rhel/docs/api/v2 && \
    ./scripts/mergeswag.sh generated/api/v2 2 >image/rhel/docs/api/v2/swagger.json

RUN make copy-go-binaries-to-image-dir


FROM registry.access.redhat.com/ubi8/nodejs-18:latest AS ui-builder

WORKDIR /go/src/github.com/stackrox/rox/app

COPY --chown=default . .

# This installs yarn from Cachi2 and makes `yarn` executable available.
# Not using `npm install --global` because it won't get us `yarn` globally.
RUN cd .konflux/bootstrap-yarn && \
    npm ci --no-audit --no-fund
ENV PATH="$PATH:/go/src/github.com/stackrox/rox/app/.konflux/bootstrap-yarn/node_modules/.bin/"

# This sets branding during UI build time. This is to make sure UI is branded as commercial RHACS (not StackRox).
# ROX_PRODUCT_BRANDING is also set in the resulting image so that Central Go code knows its RHACS.
ENV ROX_PRODUCT_BRANDING="RHACS_BRANDING"

# UI build is not hermetic because Cachi2 does not support pulling packages according to V1 of yarn.lock.
# TODO(ROX-20723): enable yarn package prefetch and make UI builds hermetic.
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
COPY --from=go-builder /go/src/github.com/stackrox/rox/app/image/rhel/static-bin/* /stackrox/
RUN GOARCH=$(uname -m) ; \
    case $GOARCH in x86_64) GOARCH=amd64 ;; aarch64) GOARCH=arm64 ;; esac ; \
    ln -s /assets/downloads/cli/roxctl-linux-$GOARCH /stackrox/roxctl ; \
    ln -s /assets/downloads/cli/roxctl-linux-$GOARCH /assets/downloads/cli/roxctl-linux

ARG MAIN_IMAGE_TAG

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
    source-location="https://github.com/stackrox/stackrox" \
    summary="Main Image for Red Hat Advanced Cluster Security for Kubernetes" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    vendor="Red Hat, Inc." \
    # We must set version label to prevent inheriting value set in the base stage.
    version="${MAIN_IMAGE_TAG}" \
    # Release label is required by EC although has no practical semantics.
    # We also set it to not inherit one from a base stage in case it's RHEL or UBI.
    release="1"

EXPOSE 8443

# TODO(ROX-22245): set proper image flavor for user-facing GA Fast Stream images.
ENV PATH="/stackrox:$PATH" \
    ROX_ROXCTL_IN_MAIN_IMAGE="true" \
    ROX_IMAGE_FLAVOR="development_build" \
    ROX_PRODUCT_BRANDING="RHACS_BRANDING"

COPY .konflux/stackrox-data/external-networks/external-networks.zip /stackrox/static-data/external-networks/external-networks.zip

COPY --from=go-builder /go/src/github.com/stackrox/rox/app/image/rhel/docs/api/v1/swagger.json /stackrox/static-data/docs/api/v1/swagger.json
COPY --from=go-builder /go/src/github.com/stackrox/rox/app/image/rhel/docs/api/v2/swagger.json /stackrox/static-data/docs/api/v2/swagger.json

# The following paths are written to in Central.
RUN chown -R 4000:4000 /etc/pki /etc/ssl && save-dir-contents /etc/pki/ca-trust /etc/ssl && \
    mkdir -p /var/lib/stackrox && chown -R 4000:4000 /var/lib/stackrox && \
    mkdir -p /var/log/stackrox && chown -R 4000:4000 /var/log/stackrox && \
    mkdir -p /var/cache/stackrox && chown -R 4000:4000 /var/cache/stackrox && \
    chown -R 4000:4000 /tmp

USER 4000:4000

FROM registry.access.redhat.com/ubi8/nodejs-18:latest AS ui-builder

WORKDIR /go/src/github.com/stackrox/rox/app

COPY . .

# This installs yarn from Cachi2 and makes `yarn` executable available.
# Not using `npm install --global` because it won't get us `yarn` globally.
RUN cd scripts/konflux/bootstrap-yarn && \
    npm ci --no-audit --no-fund
ENV PATH="$PATH:/go/src/github.com/stackrox/rox/app/scripts/konflux/bootstrap-yarn/node_modules/.bin/"

# This sets branding during UI build time. This is to make sure UI is branded as commercial RHACS (not StackRox).
# ROX_PRODUCT_BRANDING is also set in the resulting image so that Central Go code knows its RHACS.
# TODO(ROX-22338): switch branding to RHACS_BRANDING when intermediate Konflux repos aren't public.
ENV ROX_PRODUCT_BRANDING="STACKROX_BRANDING"

# UI build is not hermetic because Cachi2 does not support pulling packages according to V1 of yarn.lock.
# TODO(ROX-20723): enable yarn package prefetch and make UI builds hermetic.
RUN make -C ui build


# TODO(ROX-20312): we can't pin image tag or digest because currently there's no mechanism to auto-update that.
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest AS final-base

# TODO(ROX-20651): use content sets instead of subscription manager for access to RHEL RPMs once available.
FROM registry.access.redhat.com/ubi8/ubi:latest AS rpm-installer

ARG FINAL_STAGE_PATH="/mnt/final"

COPY --from=final-base / "$FINAL_STAGE_PATH"

COPY ./scripts/konflux/subscription-manager/* /tmp/.konflux/
RUN /tmp/.konflux/subscription-manager-bro.sh register "$FINAL_STAGE_PATH"

RUN dnf -y --installroot="$FINAL_STAGE_PATH" upgrade --nobest && \
    dnf -y --installroot="$FINAL_STAGE_PATH" module enable postgresql:13 && \
    # find is used in /stackrox/import-additional-cas \
    # snappy provides libsnappy.so.1, which is needed by most stackrox binaries \
    dnf -y --installroot="$FINAL_STAGE_PATH" install findutils snappy zstd postgresql && \
    # We can do usual cleanup while we're here: remove packages that would trigger violations. \
    dnf -y --installroot="$FINAL_STAGE_PATH" clean all && \
    rpm --root="$FINAL_STAGE_PATH" --verbose -e --nodeps $(rpm --root="$FINAL_STAGE_PATH" -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf "$FINAL_STAGE_PATH/var/cache/dnf" "$FINAL_STAGE_PATH/var/cache/yum"

RUN /tmp/.konflux/subscription-manager-bro.sh cleanup


FROM scratch

COPY --from=rpm-installer /mnt/final /

COPY --from=ui-builder /go/src/github.com/stackrox/rox/app/ui/build /ui/

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
    # TODO(ROX-20236): release label is required by EC, figure what to put in the release version on rebuilds.
    release="0" \
    source-location="https://github.com/stackrox/stackrox" \
    summary="Main Image for RHACS" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    vendor="Red Hat, Inc." \
    # We must set version label to prevent inheriting value set in the base stage.
    # TODO(ROX-20236): configure injection of dynamic version value when it becomes possible.
    version="0.0.1-todo"

EXPOSE 8443

# TODO(ROX-22245): set proper image flavor for Fast Stream images.
# TODO(ROX-22338): switch branding to RHACS_BRANDING when intermediate Konflux repos aren't public.
ENV PATH="/stackrox:$PATH" \
    ROX_ROXCTL_IN_MAIN_IMAGE="true" \
    ROX_IMAGE_FLAVOR="rhacs" \
    ROX_PRODUCT_BRANDING="STACKROX_BRANDING"

USER 4000:4000

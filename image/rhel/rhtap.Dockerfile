FROM registry.access.redhat.com/ubi8/nodejs-18:latest AS ui-builder

# Switch to root because ubi8/nodejs image runs as non-root user by default which does not let install RPMs.
USER 0:0

WORKDIR /go/src/github.com/stackrox/rox/app

COPY . .

# This sets branding during UI build time. This is to make sure UI is branded as commercial RHACS (not StackRox).
# ROX_PRODUCT_BRANDING is also set in the resulting image so that Central Go code knows its RHACS.
ENV ROX_PRODUCT_BRANDING="RHACS_BRANDING"

# This installs yarn from Cachi2 and makes `yarn` executable available.
# Not using `npm install --global` because it won't get us `yarn` globally.
RUN cd image/rhel/rhtap-bootstrap-yarn && \
    npm ci --no-audit --no-fund
ENV PATH="$PATH:/go/src/github.com/stackrox/rox/app/image/rhel/rhtap-bootstrap-yarn/node_modules/.bin/"

# UI build is not hermetic because Cachi2 does not support pulling packages according to yarn.lock yet.
# TODO(ROX-20723): make UI builds hermetic when Cachi2 supports that.
RUN make -C ui build


FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

# find and ca-certificates are used in /stackrox/import-additional-cas
# snappy provides libsnappy.so.1, which is needed by most stackrox binaries
RUN microdnf upgrade -y --nobest && \
    microdnf module enable -y postgresql:13 && \
    microdnf install -y ca-certificates findutils snappy zstd postgresql && \
    microdnf clean all && \
    rpm --verbose -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum

COPY --from=ui-builder /go/src/github.com/stackrox/rox/app/ui/build /ui/

LABEL \
    com.redhat.component="rhacs-main-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="Main Image for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="Main Image for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="main" \
    io.openshift.tags="rhacs,main,stackrox" \
    maintainer="Red Hat, Inc." \
    name="rhacs-main-rhel8" \
    source-location="https://github.com/stackrox/stackrox" \
    summary="Main Image for RHACS" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    # We must set version label to prevent inheriting value set in the base stage.
    # TODO(ROX-20236): configure injection of dynamic version value when it becomes possible.
    version="0.0.1-todo"

EXPOSE 8443

ENV PATH="/stackrox:$PATH" \
    ROX_ROXCTL_IN_MAIN_IMAGE="true" \
    ROX_IMAGE_FLAVOR="rhacs" \
    ROX_PRODUCT_BRANDING="RHACS_BRANDING"

USER 4000:4000

HEALTHCHECK CMD curl --insecure --fail https://127.0.0.1:8443/v1/ping

ARG MAIN_IMAGE
FROM ${MAIN_IMAGE} AS main-image

FROM registry.access.redhat.com/ubi9/ubi-micro:latest@sha256:7e7f79ab747bf2b452e3043dd89f388e92be4c7fdcc8b815b58adf6c99c39c95

COPY --from=main-image /assets/downloads/cli/roxctl-* /tmp/cli/

RUN mkdir -p /assets/downloads/cli && \
    cd /tmp/cli && \
    find . -maxdepth 1 -type f -name 'roxctl-*' -executable | while read f; do \
        tar czf "/assets/downloads/cli/${f#./}.tar.gz" "${f#./}"; \
    done && \
    rm -rf /tmp/cli

ARG BUILD_TAG

LABEL \
    com.redhat.component="rhacs-roxctl-tars-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="Roxctl CLI tarballs for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="Roxctl CLI tarballs for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="roxctl-tars" \
    io.openshift.tags="rhacs,roxctl,stackrox" \
    maintainer="Red Hat, Inc." \
    name="advanced-cluster-security/rhacs-roxctl-tars-rhel9" \
    # Custom Snapshot creation in `operator-bundle-pipeline` depends on source-location label to be set correctly.
    source-location="https://github.com/stackrox/stackrox" \
    summary="Roxctl CLI tarballs for Red Hat Advanced Cluster Security for Kubernetes" \
    version="${BUILD_TAG}" \
    # Release label is required by EC although has no practical semantics.
    release="1"

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest AS final

USER root

LABEL \
    com.redhat.component="rhacs-central-db-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="Central Database Image for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="Central Database Image for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="central-db" \
    io.openshift.tags="rhacs,central-db,stackrox" \
    maintainer="Red Hat, Inc." \
    name="rhacs-central-db-rhel8" \
    source-location="https://github.com/stackrox/stackrox" \
    summary="Central DB for RHACS" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    # We must set version label to prevent inheriting value set in the base stage.
    # TODO(ROX-20236): configure injection of dynamic version value when it becomes possible.
    version="0.0.1-todo"

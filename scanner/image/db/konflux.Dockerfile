FROM registry.redhat.io/rhel8/postgresql-15:latest@sha256:0576206b44a5cb073b6364f7af104a4a17994f9bb8c7a948da9588d6b4bbf1e8

ARG BUILD_TAG
RUN if [[ "$BUILD_TAG" == "" ]]; then >&2 echo "error: required BUILD_TAG arg is unset"; exit 6; fi

LABEL \
    com.redhat.component="rhacs-scanner-v4-db-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="Scanner v4 Database Image for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="Scanner v4 Database Image for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="scanner-v4-db" \
    io.openshift.tags="rhacs,scanner-v4-db,stackrox" \
    maintainer="Red Hat, Inc." \
    name="rhacs-scanner-v4-db-rhel8" \
    # Custom Snapshot creation in `operator-bundle-pipeline` depends on source-location label to be set correctly.
    source-location="https://github.com/stackrox/stackrox" \
    summary="Scanner v4 DB for Red Hat Advanced Cluster Security for Kubernetes" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    # We must set version label to prevent inheriting value set in the base stage.
    version="${BUILD_TAG}" \
    # Release label is required by EC although has no practical semantics.
    # We also set it to not inherit one from a base stage in case it's RHEL or UBI.
    release="1"

USER root

COPY \
     scanner/image/db/scripts/docker-entrypoint.sh \
     scanner/image/db/scripts/init-entrypoint.sh \
     /usr/local/bin/

RUN localedef -f UTF-8 -i en_US en_US.UTF-8 && \
    mkdir -p /var/lib/postgresql && \
    groupmod -g 70 postgres && \
    usermod -u 70 postgres -d /var/lib/postgresql && \
    chown -R postgres:postgres /var/lib/postgresql && \
    chown -R postgres:postgres /var/run/postgresql && \
    dnf clean all && \
    rpm --verbose -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum

COPY LICENSE /licenses/LICENSE

ENV LANG=en_US.utf8

STOPSIGNAL SIGINT

USER 70:70

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 5432
CMD ["postgres", "-c", "config_file=/etc/stackrox.d/config/postgresql.conf"]

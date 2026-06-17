ARG PG_VERSION=15
FROM registry.redhat.io/rhel9/postgresql-${PG_VERSION}:latest@sha256:ba85fa939583ef38cbd53c815904e871dac359cef6582e8ea0efe8534fcd8093 AS final

USER root

ARG BUILD_TAG
RUN if [[ "$BUILD_TAG" == "" ]]; then >&2 echo "error: required BUILD_TAG arg is unset"; exit 6; fi

LABEL com.redhat.component=rhacs-central-db-container
LABEL com.redhat.license_terms=https://www.redhat.com/agreements
LABEL description="Central Database Image for Red Hat Advanced Cluster Security for Kubernetes"
LABEL io.k8s.description="Central Database Image for Red Hat Advanced Cluster Security for Kubernetes"
LABEL io.k8s.display-name=central-db
LABEL io.openshift.tags=rhacs,central-db,stackrox
LABEL maintainer="Red Hat, Inc."
LABEL name=advanced-cluster-security/rhacs-central-db-rhel9
# Custom Snapshot creation in `operator-bundle-pipeline` depends on source-location label to be set correctly.
LABEL source-location=https://github.com/stackrox/stackrox
LABEL summary="Central DB for Red Hat Advanced Cluster Security for Kubernetes"
LABEL url=https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c
# We must set version label to prevent inheriting value set in the base stage.
LABEL version=${BUILD_TAG}
# Release label is required by EC although has no practical semantics.
# We also set it to not inherit one from a base stage in case it's RHEL or UBI.
LABEL release=1

RUN localedef -f UTF-8 -i en_US en_US.UTF-8 && \
    mkdir -p /var/lib/postgresql && \
    groupmod -g 70 postgres && \
    usermod -u 70 postgres -d /var/lib/postgresql && \
    chown -R postgres:postgres /var/lib/postgresql && \
    chown -R postgres:postgres /var/run/postgresql && \
    # Change ownership of directories that are owned by the original postgres uid.
    # Detect these with `find / -uid 26` in the original container.
    chown -R postgres /var/lib/pgsql && chown -R postgres /opt/app-root && \
    # Cleanup
    dnf clean all && \
    rpm --verbose -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum

COPY LICENSE /licenses/LICENSE

COPY image/postgres/scripts \
    /usr/local/bin/

ENV LANG="en_US.utf8"

# Use SIGINT to bring down with Fast Shutdown mode
STOPSIGNAL SIGINT

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 5432
# Note that postgresql.conf should be mounted from ConfigMap
CMD ["postgres", "-c", "config_file=/etc/stackrox.d/config/postgresql.conf"]

HEALTHCHECK --interval=10s --timeout=5s CMD pg_isready

USER 70:70

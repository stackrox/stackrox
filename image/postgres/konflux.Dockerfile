ARG PG_VERSION=15
FROM registry.redhat.io/rhel8/postgresql-${PG_VERSION}:latest@sha256:1435c8fa9ac2297ac2912c3377bc1408f2a342c35bb249474ada675f462ae986 AS final

USER root

ARG BUILD_TAG
RUN if [[ "$BUILD_TAG" == "" ]]; then >&2 echo "error: required BUILD_TAG arg is unset"; exit 6; fi

LABEL \
    com.redhat.component="rhacs-central-db-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="Central Database Image for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="Central Database Image for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="central-db" \
    io.openshift.tags="rhacs,central-db,stackrox" \
    maintainer="Red Hat, Inc." \
    name="rhacs-central-db-rhel8" \
    # Custom Snapshot creation in `operator-bundle-pipeline` depends on source-location label to be set correctly.
    source-location="https://github.com/stackrox/stackrox" \
    summary="Central DB for Red Hat Advanced Cluster Security for Kubernetes" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    # We must set version label to prevent inheriting value set in the base stage.
    version="${BUILD_TAG}" \
    # Release label is required by EC although has no practical semantics.
    # We also set it to not inherit one from a base stage in case it's RHEL or UBI.
    release="1"

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

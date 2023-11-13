FROM registry.redhat.io/rhel8/postgresql-13:latest AS final

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

RUN dnf upgrade -y --nobest && \
    localedef -f UTF-8 -i en_US en_US.UTF-8 && \
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

USER 70:70

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

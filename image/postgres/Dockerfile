ARG PG_VERSION=15
ARG CENTOS_VERSION=c8s
FROM quay.io/sclorg/postgresql-${PG_VERSION}-${CENTOS_VERSION}:latest AS final

USER root

ARG MAIN_IMAGE_TAG

LABEL \
    com.redhat.component="rhacs-central-db-container" \
    description="Central Database Image for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="Central Database Image for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="central-db" \
    io.openshift.tags="rhacs,central-db,stackrox" \
    maintainer="Red Hat, Inc." \
    name="rhacs-central-db-rhel8" \
    source-location="https://github.com/stackrox/stackrox" \
    summary="Central DB for Red Hat Advanced Cluster Security for Kubernetes" \
    # We must set version label to prevent inheriting value set in the base stage.
    version="${MAIN_IMAGE_TAG}" \
    # Release label is required by EC although has no practical semantics.
    # We also set it to not inherit one from a base stage in case it's RHEL or UBI.
    release="1"

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
    rm -rf /var/cache/dnf /var/cache/yum && \
    mkdir /docker-entrypoint-initdb.d

COPY scripts \
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

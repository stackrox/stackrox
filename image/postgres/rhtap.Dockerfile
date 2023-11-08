FROM registry.access.redhat.com/ubi8/ubi-minimal:latest as builder

COPY . /stackrox

FROM registry.redhat.io/rhel8/postgresql-13 as final

USER root

LABEL \
    com.redhat.component="rhacs-central-db-container" \
    name="rhacs-central-db-rhel8" \
    maintainer="Red Hat, Inc." \
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

COPY --from=builder --chown=postgres:postgres \
    /stackrox/image/postgres/scripts/docker-entrypoint.sh \
    /usr/local/bin/
COPY --from=builder \
    /stackrox/image/postgres/scripts/init-entrypoint.sh \
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

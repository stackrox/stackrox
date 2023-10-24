ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8-minimal
ARG RPMS_BASE_IMAGE=ubi8
ARG BASE_TAG=latest

FROM ${BASE_REGISTRY}/${RPMS_BASE_IMAGE}:${BASE_TAG} AS postgres_rpms

COPY download.sh /download.sh
RUN /download.sh

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

LABEL name="scanner-db" \
      vendor="StackRox" \
      maintainer="https://stackrox.io/" \
      summary="Static vulnerability scanner database for the StackRox Security Platform" \
      description="This image supports static vulnerability scanning for the StackRox Security Platform."

# If this is updated, be sure to update the version of the postgres server installed below.
ENV PG_MAJOR=13
ENV PATH="$PATH:/usr/pgsql-$PG_MAJOR/bin/" \
    PGDATA="/var/lib/postgresql/data/pgdata"
ENV LANG=en_US.utf8

COPY signatures/RPM-GPG-KEY-PGDG-13 /
COPY scripts/docker-entrypoint.sh /usr/local/bin/
COPY scripts/init-entrypoint.sh /usr/local/bin/
COPY --from=postgres_rpms /rpms/postgres.rpm /rpms/postgres-libs.rpm /rpms/postgres-server.rpm /rpms/postgres-contrib.rpm /tmp/

RUN microdnf upgrade --nobest && \
    # groupadd is in shadow-utils package that is not installed by default.
    microdnf install shadow-utils && \
    groupadd -g 70 postgres && \
    adduser postgres -u 70 -g 70 -d /var/lib/postgresql -s /bin/sh && \
    microdnf install \
        ca-certificates libicu systemd-sysv \
        glibc-locale-source glibc-langpack-en \
        perl-libs libxslt && \
    rpm -i /tmp/postgres-libs.rpm /tmp/postgres-server.rpm /tmp/postgres.rpm /tmp/postgres-contrib.rpm && \
    # Restore /usr/share/zoneinfo that's empty in ubi-minimal because postgres reads timezone data from it.
    # https://access.redhat.com/solutions/5616681
    microdnf reinstall tzdata && \
    microdnf clean all && \
    # (Optional) Remove line below to keep package management utilities
    rpm -e --nodeps $(rpm -qa shadow-utils curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum /tmp/postgres-libs.rpm /tmp/postgres-server.rpm /tmp/postgres.rpm /tmp/postgres-contrib.rpm && \
    localedef -f UTF-8 -i en_US en_US.UTF-8 && \
    chown postgres:postgres /usr/local/bin/docker-entrypoint.sh && \
    chmod +x /usr/local/bin/docker-entrypoint.sh && \
    mkdir /docker-entrypoint-initdb.d

STOPSIGNAL SIGINT

# This is equivalent to postgres:postgres.
USER 70:70

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 5432
CMD ["postgres", "-c", "config_file=/etc/stackrox.d/config/postgresql.conf"]

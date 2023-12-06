ARG RPMS_REGISTRY=registry.access.redhat.com
ARG RPMS_BASE_IMAGE=ubi8
ARG RPMS_BASE_TAG=latest
ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8-minimal
ARG BASE_TAG=latest

FROM ${RPMS_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} AS postgres_rpms

COPY download.sh /download.sh
RUN /download.sh

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

LABEL name="scanner-db" \
      vendor="StackRox" \
      maintainer="https://stackrox.io/" \
      summary="Static vulnerability scanner database for the StackRox Security Platform" \
      description="This image supports static vulnerability scanning for the StackRox Security Platform."

# If this is updated, be sure to update the version of the postgres server installed below.
ENV PG_MAJOR=15
ENV PATH="$PATH:/usr/pgsql-$PG_MAJOR/bin/" \
    PGDATA="/var/lib/postgresql/data/pgdata"
ENV LANG=en_US.utf8

COPY signatures/RPM-GPG-KEY-PGDG-15 /
COPY scripts/docker-entrypoint.sh scripts/init-entrypoint.sh /usr/local/bin/
COPY --from=postgres_rpms /rpms/postgres.rpm /rpms/postgres-libs.rpm /rpms/postgres-server.rpm /rpms/postgres-contrib.rpm /tmp/

RUN microdnf upgrade -y --nobest && \
    # groupadd is in shadow-utils package that is not installed by default.
    microdnf install -y shadow-utils && \
    groupadd -g 70 postgres && \
    adduser postgres -u 70 -g 70 -d /var/lib/postgresql -s /bin/sh && \
    microdnf install -y \
        ca-certificates libicu systemd-sysv \
        glibc-locale-source glibc-langpack-en \
        perl-libs libxslt python3 lz4 && \
    if [[ $(awk -F'=' '/VERSION_ID/{ gsub(/"/,""); print substr($2,1,1)}' /etc/os-release) -gt 8 ]]; then microdnf install -y uuid; fi && \
    rpm -i /tmp/postgres-libs.rpm /tmp/postgres-server.rpm /tmp/postgres.rpm /tmp/postgres-contrib.rpm && \
    # Restore /usr/share/zoneinfo that's empty in ubi-minimal because postgres reads timezone data from it.
    # https://access.redhat.com/solutions/5616681
    microdnf reinstall -y tzdata && \
    microdnf clean all && \
    # (Optional) Remove line below to keep package management utilities
    rpm -e --nodeps $(rpm -qa shadow-utils curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum /tmp/postgres-libs.rpm /tmp/postgres-server.rpm /tmp/postgres.rpm /tmp/postgres-contrib.rpm && \
    localedef -f UTF-8 -i en_US en_US.UTF-8 && \
    mkdir /docker-entrypoint-initdb.d

STOPSIGNAL SIGINT

# This is equivalent to postgres:postgres.
USER 70:70

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 5432
CMD ["postgres", "-c", "config_file=/etc/stackrox.d/config/postgresql.conf"]

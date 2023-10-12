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

LABEL name="central-db" \
      vendor="StackRox" \
      maintainer="https://stackrox.io/" \
      summary="Stackrox Central Database for the StackRox Kubernetes Security Platform" \
      description="This image provides Database services to Stackrox Central in the StackRox Kubernetes Security Platform."

ENV PG_MAJOR=13
ENV PATH="$PATH:/usr/pgsql-$PG_MAJOR/bin/"
ENV LANG=en_US.utf8

COPY signatures/RPM-GPG-KEY-PGDG-13 /
COPY scripts/docker-entrypoint.sh /usr/local/bin/
COPY scripts/init-entrypoint.sh /usr/local/bin/
COPY --from=postgres_rpms /rpms/postgres.rpm /rpms/postgres-libs.rpm /rpms/postgres-server.rpm /rpms/postgres-contrib.rpm /tmp/

RUN microdnf upgrade -y --nobest && \
    # groupadd is in shadow-utils package that is not installed by default.
    microdnf -y --setopt=install_weak_deps=0 --nodocs install shadow-utils && \
    groupadd -g 70 postgres && \
    adduser postgres -u 70 -g 70 -d /var/lib/postgresql -s /bin/sh && \
    rpm --import RPM-GPG-KEY-PGDG-13 && \
    microdnf -y --setopt=install_weak_deps=0 --nodocs install \
        ca-certificates libicu systemd-sysv \
        glibc-locale-source glibc-langpack-en \
        perl-libs libxslt python3 uuid && \
    rpm -i /tmp/postgres-libs.rpm /tmp/postgres-server.rpm /tmp/postgres.rpm /tmp/postgres-contrib.rpm && \
    # Restore /usr/share/zoneinfo that's empty in ubi-minimal because postgres reads timezone data from it.
    # https://access.redhat.com/solutions/5616681
    microdnf -y --setopt=install_weak_deps=0 --nodocs reinstall tzdata && \
    microdnf clean all -y && \
    rpm -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum && \
    localedef -f UTF-8 -i en_US en_US.UTF-8 && \
    chown postgres:postgres /usr/local/bin/docker-entrypoint.sh && \
    chmod +x /usr/local/bin/docker-entrypoint.sh && \
    mkdir /docker-entrypoint-initdb.d

# Use SIGINT to bring down with Fast Shutdown mode
STOPSIGNAL SIGINT

USER postgres:postgres

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 5432
CMD ["postgres", "-c", "config_file=/etc/stackrox.d/config/postgresql.conf"]

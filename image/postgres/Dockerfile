ARG PG_VERSION=15

FROM registry.access.redhat.com/ubi9:latest AS postgres_rpms

COPY download.sh /download.sh
RUN bash /download.sh

FROM registry.access.redhat.com/ubi9/ubi-micro:latest AS ubi-micro-base
FROM registry.access.redhat.com/ubi9:latest AS package_installer

COPY --from=ubi-micro-base / /out/

RUN dnf install \
        --installroot=/out/ \
        --releasever=9 \
        --setopt=install_weak_deps=0 \
        --nodocs \
        --allowerasing \
        -y \
        findutils util-linux shadow-utils bash coreutils ca-certificates \
        openldap glibc-langpack-en glibc-locale-source libicu libxslt lz4 \
        perl-libs tar uuid zstd && \
    dnf reinstall \
        --installroot=/out/ \
        --releasever=9 \
        --setopt=reposdir=/etc/yum.repos.d \
        -y tzdata && \
    dnf clean all --installroot=/out/ && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum

COPY --from=postgres_rpms \
    /rpms/postgresql15.rpm /rpms/postgresql15-libs.rpm /rpms/postgresql15-server.rpm /rpms/postgresql15-contrib.rpm \
    /rpms/postgresql-upgrade.rpm /tmp/
COPY signatures/RPM-GPG-KEY-PGDG-13 /tmp/PGDG-RPM-GPG-KEY-RHEL

RUN rpm --root=/out/ --import /tmp/PGDG-RPM-GPG-KEY-RHEL && \
    rpm --root=/out/ -ivh --nodeps \
        /tmp/postgresql15-libs.rpm /tmp/postgresql15-server.rpm \
        /tmp/postgresql15.rpm /tmp/postgresql15-contrib.rpm /tmp/postgresql-upgrade.rpm && \
    rm -rf /tmp/*.rpm /tmp/PGDG-RPM-GPG-KEY-RHEL

FROM ubi-micro-base AS final

USER root

ARG MAIN_IMAGE_TAG
ARG QUAY_TAG_EXPIRATION

LABEL \
    com.redhat.component="rhacs-central-db-container" \
    description="Central Database Image for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="Central Database Image for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="central-db" \
    io.openshift.tags="rhacs,central-db,stackrox" \
    maintainer="Red Hat, Inc." \
    name="rhacs-central-db-rhel9" \
    source-location="https://github.com/stackrox/stackrox" \
    summary="Central DB for Red Hat Advanced Cluster Security for Kubernetes" \
    version="${MAIN_IMAGE_TAG}" \
    release="1" \
    quay.expires-after="${QUAY_TAG_EXPIRATION}"

ENV PG_MAJOR=15 \
    POSTGRESQL_PREV_VERSION=13 \
    PATH="$PATH:/usr/pgsql-15/bin/" \
    PGDATA="/var/lib/postgresql/data/pgdata" \
    LANG="en_US.utf8"

COPY --from=package_installer /out/ /
COPY --link scripts /usr/local/bin/

RUN localedef -f UTF-8 -i en_US en_US.UTF-8 && \
    if [ ! -e /usr/pgsql-15/bin ]; then mkdir -p /usr/pgsql-15 && ln -s /usr/bin /usr/pgsql-15/bin; fi && \
    if [ ! -e /usr/pgsql-13/bin ]; then mkdir -p /usr/pgsql-13 && ln -s /usr/lib64/pgsql/postgresql-13/bin /usr/pgsql-13/bin; fi && \
    if [ ! -e /usr/lib64/libpq.so.private15-5 ]; then \
        if [ -e /usr/pgsql-15/lib/libpq.so.5 ]; then \
            ln -s /usr/pgsql-15/lib/libpq.so.5 /usr/lib64/libpq.so.private15-5; \
        elif [ -e /usr/lib64/libpq.so.5 ]; then \
            ln -s /usr/lib64/libpq.so.5 /usr/lib64/libpq.so.private15-5; \
        fi; \
    fi && ldconfig && \
    if getent group postgres >/dev/null; then groupmod -g 70 postgres; else groupadd -g 70 postgres; fi && \
    if id -u postgres >/dev/null 2>&1; then usermod -u 70 -g 70 -d /var/lib/postgresql postgres; else useradd -u 70 -g 70 -d /var/lib/postgresql -s /bin/sh postgres; fi && \
    mkdir -p /docker-entrypoint-initdb.d /var/run/postgresql /var/lib/postgresql && \
    if [ -d /var/lib/pgsql ]; then chown -R postgres:postgres /var/lib/pgsql; fi && \
    chown -R postgres:postgres /var/lib/postgresql /var/run/postgresql && \
    chmod 03775 /var/run/postgresql

STOPSIGNAL SIGINT

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 5432
CMD ["postgres", "-c", "config_file=/etc/stackrox.d/config/postgresql.conf"]

HEALTHCHECK --interval=10s --timeout=5s CMD pg_isready

# pg_upgrade requires read and write access to the current directory.
WORKDIR /var/lib/postgresql

USER 70:70

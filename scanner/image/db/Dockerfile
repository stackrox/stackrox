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
        perl-libs uuid zstd && \
    dnf reinstall \
        --installroot=/out/ \
        --releasever=9 \
        --setopt=reposdir=/etc/yum.repos.d \
        -y tzdata && \
    dnf clean all --installroot=/out/ && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum

COPY --from=postgres_rpms /rpms/postgres.rpm /rpms/postgres-libs.rpm /rpms/postgres-server.rpm /rpms/postgres-contrib.rpm /tmp/
COPY signatures/RPM-GPG-KEY-PGDG-13 /tmp/PGDG-RPM-GPG-KEY-RHEL

RUN rpm --root=/out/ --import /tmp/PGDG-RPM-GPG-KEY-RHEL && \
    rpm --root=/out/ -ivh --nodeps \
        /tmp/postgres-libs.rpm /tmp/postgres-server.rpm \
        /tmp/postgres.rpm /tmp/postgres-contrib.rpm && \
    rm -rf /tmp/*.rpm /tmp/PGDG-RPM-GPG-KEY-RHEL

FROM ubi-micro-base

USER root

ENV PG_MAJOR=15 \
    PATH="$PATH:/usr/pgsql-15/bin/" \
    PGDATA="/var/lib/postgresql/data/pgdata" \
    LANG="en_US.utf8"

# This will be ignored if empty in the init script.
COPY --link init-bundles/db-init.dump.zst /db-init.dump.zst
COPY --link scripts/docker-entrypoint.sh scripts/init-entrypoint.sh scripts/cert-watcher.sh /usr/local/bin/
COPY --from=package_installer /out/ /

ARG LABEL_VERSION
ARG LABEL_RELEASE
ARG QUAY_TAG_EXPIRATION

LABEL name="scanner-v4-db" \
      vendor="StackRox" \
      maintainer="https://stackrox.io/" \
      summary="Static vulnerability scanner database for the StackRox Security Platform" \
      description="This image supports static vulnerability scanning for the StackRox Security Platform." \
      version="${LABEL_VERSION}" \
      release="${LABEL_RELEASE}" \
      quay.expires-after="${QUAY_TAG_EXPIRATION}"

RUN localedef -f UTF-8 -i en_US en_US.UTF-8 && \
    if [ ! -e /usr/pgsql-15/bin ]; then mkdir -p /usr/pgsql-15 && ln -s /usr/bin /usr/pgsql-15/bin; fi && \
    if getent group postgres >/dev/null; then groupmod -g 70 postgres; else groupadd -g 70 postgres; fi && \
    if id -u postgres >/dev/null 2>&1; then usermod -u 70 -g 70 -d /var/lib/postgresql postgres; else useradd -u 70 -g 70 -d /var/lib/postgresql -s /bin/sh postgres; fi && \
    mkdir -p /docker-entrypoint-initdb.d /var/run/postgresql /var/lib/postgresql && \
    if [ -d /var/lib/pgsql ]; then chown -R postgres:postgres /var/lib/pgsql; fi && \
    chown -R postgres:postgres /var/lib/postgresql /var/run/postgresql && \
    chmod 03775 /var/run/postgresql

STOPSIGNAL SIGINT

USER 70:70

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 5432
CMD ["postgres", "-c", "config_file=/etc/stackrox.d/config/postgresql.conf"]

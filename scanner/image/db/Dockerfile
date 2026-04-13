FROM registry.access.redhat.com/ubi9:latest AS ubi-base
FROM registry.access.redhat.com/ubi9/ubi-micro:latest AS ubi-micro-base

FROM ubi-base AS postgres_rpms

COPY scripts/download.sh /download.sh
RUN /download.sh

FROM ubi-base AS package_installer

COPY --from=ubi-micro-base / /out/

RUN dnf install \
    --installroot=/out/ \
    --releasever=9 \
    --setopt=install_weak_deps=0 \
    --nodocs \
    -y \
    shadow-utils \
    ca-certificates \
    openldap \
    glibc-langpack-en \
    glibc-locale-source \
    libicu \
    libxslt \
    lz4 \
    perl-libs \
    python3 \
    systemd-sysv \
    uuid \
    zstd && \
    # Reinstall tzdata without --nodocs to populate /usr/share/zoneinfo directory.
    # ubi9-micro ships with tzdata pre-installed but /usr/share/zoneinfo is empty.
    # Regular install won't work since package is already present, must use reinstall.
    # PostgreSQL requires timezone files at runtime.
    dnf reinstall \
        --installroot=/out/ \
        --releasever=9 \
        --setopt=reposdir=/etc/yum.repos.d \
        -y \
        tzdata && \
    dnf clean all --installroot=/out/ && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum

COPY --from=postgres_rpms /rpms/postgres.rpm /rpms/postgres-libs.rpm /rpms/postgres-server.rpm /rpms/postgres-contrib.rpm /tmp/
COPY signatures/PGDG-RPM-GPG-KEY-RHEL /tmp/

RUN rpm --root=/out/ --import /tmp/PGDG-RPM-GPG-KEY-RHEL && \
    rpm --root=/out/ -ivh --nodeps /tmp/postgres-libs.rpm /tmp/postgres-server.rpm /tmp/postgres.rpm /tmp/postgres-contrib.rpm && \
    rm -rf /tmp/*.rpm /tmp/PGDG-RPM-GPG-KEY-RHEL

FROM ubi-micro-base

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

# If this is updated, be sure to update postgres_major in download.sh and the signature file.
ENV PG_MAJOR=15
ENV PATH="$PATH:/usr/pgsql-$PG_MAJOR/bin/"
ENV LANG=en_US.utf8

COPY --from=package_installer /out/ /

COPY init-bundles/db-init.dump.zst /db-init.dump.zst
COPY scripts/docker-entrypoint.sh scripts/init-entrypoint.sh /usr/local/bin/

RUN \
    groupadd -g 70 postgres && \
    useradd postgres -u 70 -g 70 -d /var/lib/postgresql -s /bin/sh && \
    localedef -f UTF-8 -i en_US en_US.UTF-8 && \
    mkdir -p /docker-entrypoint-initdb.d /var/run/postgresql && \
    chown postgres:postgres /var/run/postgresql && \
    chmod 03775 /var/run/postgresql

STOPSIGNAL SIGINT

# This is equivalent to postgres:postgres.
USER 70:70

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 5432
CMD ["postgres", "-c", "config_file=/etc/stackrox.d/config/postgresql.conf"]

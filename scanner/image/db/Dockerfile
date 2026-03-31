ARG RPMS_REGISTRY=registry.access.redhat.com
ARG RPMS_BASE_IMAGE=ubi9
ARG RPMS_BASE_TAG=latest

ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi9-micro
ARG BASE_TAG=latest

FROM ${RPMS_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} AS postgres_rpms

COPY scripts/download.sh /download.sh
RUN /download.sh

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG} AS ubi-micro-base

FROM ${RPMS_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} AS dependency_builder

COPY --from=ubi-micro-base / /out/

# If this is updated, be sure to update postgres_major in download.sh and the signature file.
ENV PG_MAJOR=15

RUN dnf install \
    --installroot=/out/ \
    --releasever=9 \
    --setopt=install_weak_deps=0 \
    --nodocs \
    -y \
    findutils \
    util-linux \
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
    if [[ $(awk -F'=' '/VERSION_ID/{ gsub(/"/,""); print substr($2,1,1)}' /etc/os-release) -gt 8 ]]; then \
        dnf install --installroot=/out/ --releasever=8 --setopt=install_weak_deps=0 --nodocs -y uuid; \
    fi && \
    dnf --installroot=/out/ clean all && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum

COPY --from=postgres_rpms /rpms/postgres.rpm /rpms/postgres-libs.rpm /rpms/postgres-server.rpm /rpms/postgres-contrib.rpm /tmp/

COPY signatures/PGDG-RPM-GPG-KEY-RHEL /tmp/
RUN rpm --root=/out/ --import /tmp/PGDG-RPM-GPG-KEY-RHEL && \
    rpm --root=/out/ -ivh --nodeps /tmp/postgres-libs.rpm /tmp/postgres-server.rpm /tmp/postgres.rpm /tmp/postgres-contrib.rpm && \
    rm -rf /tmp/*.rpm /tmp/PGDG-RPM-GPG-KEY-RHEL

RUN chroot /out /bin/sh -c " \
    if getent group postgres >/dev/null; then \
        current_gid=\$(getent group postgres | cut -d: -f3); \
        if [ \$current_gid -ne 70 ]; then \
            groupmod -g 70 postgres; \
        fi; \
    else \
        groupadd -g 70 postgres; \
    fi && \
    if id -u postgres &>/dev/null; then \
        current_uid=\$(id -u postgres); \
        if [ \$current_uid -ne 70 ]; then \
            usermod -u 70 -g 70 postgres; \
        fi; \
    else \
        useradd postgres -u 70 -g 70 -d /var/lib/postgresql -s /bin/sh; \
    fi && \
    localedef -f UTF-8 -i en_US en_US.UTF-8 \
    "

RUN mkdir -p /out/docker-entrypoint-initdb.d \
             /out/var/run/postgresql && \
    chroot /out /bin/sh -c "chown postgres:postgres /var/run/postgresql && chmod 03775 /var/run/postgresql"

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

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

ENV PG_MAJOR=15
ENV PATH="$PATH:/usr/pgsql-$PG_MAJOR/bin/"
ENV LANG=en_US.utf8

COPY --from=dependency_builder /out/ /

COPY init-bundles/db-init.dump.zst /db-init.dump.zst
COPY scripts/docker-entrypoint.sh scripts/init-entrypoint.sh /usr/local/bin/

STOPSIGNAL SIGINT

# This is equivalent to postgres:postgres.
USER 70:70

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 5432
CMD ["postgres", "-c", "config_file=/etc/stackrox.d/config/postgresql.conf"]

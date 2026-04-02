ARG RPMS_REGISTRY=registry.access.redhat.com
ARG RPMS_BASE_IMAGE=ubi9
ARG RPMS_BASE_TAG=latest

ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi9-micro
ARG BASE_TAG=latest

ARG PG_VERSION=15

FROM ${RPMS_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} AS postgres_rpms

COPY download.sh /download.sh
RUN /download.sh

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG} AS ubi-micro-base

FROM ${RPMS_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} AS dependency_builder

COPY --from=ubi-micro-base / /out/

ENV PG_MAJOR=15

RUN dnf install \
    --installroot=/out/ \
    --releasever=9 \
    --setopt=install_weak_deps=0 \
    --nodocs \
    --allowerasing \
    -y \
    findutils \
    util-linux \
    shadow-utils \
    bash \
    coreutils \
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
    dnf reinstall \
        --installroot=/out/ \
        --releasever=9 \
        --setopt=reposdir=/etc/yum.repos.d \
        -y \
        tzdata && \
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
             /out/var/run/postgresql \
             /out/var/lib/postgresql && \
    chroot /out /bin/sh -c "chown postgres:postgres /var/run/postgresql && chmod 03775 /var/run/postgresql && \
                             chown -R postgres:postgres /var/lib/postgresql"

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG} AS final

USER root

ARG MAIN_IMAGE_TAG

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
    release="1"

ENV PG_MAJOR=15
ENV PATH="$PATH:/usr/pgsql-$PG_MAJOR/bin/"
ENV LANG="en_US.utf8"
ENV PGDATA="/var/lib/postgresql/data"

COPY --from=dependency_builder /out/ /

COPY scripts/docker-entrypoint.sh scripts/init-entrypoint.sh /usr/local/bin/

STOPSIGNAL SIGINT

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 5432
CMD ["postgres", "-c", "config_file=/etc/stackrox.d/config/postgresql.conf"]

HEALTHCHECK --interval=10s --timeout=5s CMD pg_isready

USER 70:70

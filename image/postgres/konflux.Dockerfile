ARG PG_VERSION=15

FROM registry.access.redhat.com/ubi9/ubi-micro:latest@sha256:f332c99eb8f798a8486821c91937f10ad64ee83d7e739303be2df051040918f6 AS ubi-micro-base

FROM registry.access.redhat.com/ubi9/ubi:latest@sha256:25a147defd01e19674714f55d17538c8dbe55d8c305fa157ecc3f9c8977b05b6 AS package_installer

ARG PG_VERSION

# Copy ubi-micro base to /out/ to preserve its rpmdb.
COPY --from=ubi-micro-base / /out/

RUN dnf module enable -y \
        --installroot=/out/ \
        --setopt=reposdir=/etc/yum.repos.d \
        --releasever=9 \
        postgresql:${PG_VERSION} && \
    dnf install -y \
        --installroot=/out/ \
        --setopt=reposdir=/etc/yum.repos.d \
        --releasever=9 \
        --setopt=install_weak_deps=0 \
        --nodocs \
        bash ca-certificates findutils glibc-langpack-en \
        glibc-locale-source gzip less libicu libxslt lz4 openldap openssl \
        perl-libs postgresql postgresql-contrib postgresql-server python3 \
        shadow-utils systemd-sysv tar tzdata util-linux uuid zstd && \
    dnf reinstall -y \
        --installroot=/out/ \
        --setopt=reposdir=/etc/yum.repos.d \
        --releasever=9 \
        tzdata && \
    dnf clean all --installroot=/out/ && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum

FROM ubi-micro-base

USER root

ARG BUILD_TAG
RUN if [[ "$BUILD_TAG" == "" ]]; then >&2 echo "error: required BUILD_TAG arg is unset"; exit 6; fi

LABEL \
    com.redhat.component="rhacs-central-db-container" \
    com.redhat.license_terms="https://www.redhat.com/agreements" \
    description="Central Database Image for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.description="Central Database Image for Red Hat Advanced Cluster Security for Kubernetes" \
    io.k8s.display-name="central-db" \
    io.openshift.tags="rhacs,central-db,stackrox" \
    maintainer="Red Hat, Inc." \
    name="advanced-cluster-security/rhacs-central-db-rhel9" \
    # Custom Snapshot creation in `operator-bundle-pipeline` depends on source-location label to be set correctly.
    source-location="https://github.com/stackrox/stackrox" \
    summary="Central DB for Red Hat Advanced Cluster Security for Kubernetes" \
    url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c" \
    # We must set version label to prevent inheriting value set in the base stage.
    version="${BUILD_TAG}" \
    # Release label is required by EC although has no practical semantics.
    # We also set it to not inherit one from a base stage in case it's RHEL or UBI.
    release="1"

COPY --from=package_installer /out/ /

RUN localedef -f UTF-8 -i en_US en_US.UTF-8 && \
    groupmod -g 70 postgres && \
    usermod -u 70 postgres -d /var/lib/postgresql && \
    mkdir -p /var/lib/postgresql /var/run/postgresql && \
    chown -R postgres:postgres /var/lib/postgresql /var/run/postgresql && \
    if [ -d /var/lib/pgsql ]; then chown -R postgres /var/lib/pgsql; fi && \
    if [ -d /opt/app-root ]; then chown -R postgres /opt/app-root; fi

COPY LICENSE /licenses/LICENSE

COPY image/postgres/scripts \
    /usr/local/bin/

ENV PG_MAJOR=15 \
    PGDATA="/var/lib/postgresql/data/pgdata" \
    LANG="en_US.utf8"

# Use SIGINT to bring down with Fast Shutdown mode
STOPSIGNAL SIGINT

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 5432
# Note that postgresql.conf should be mounted from ConfigMap
CMD ["postgres", "-c", "config_file=/etc/stackrox.d/config/postgresql.conf"]

HEALTHCHECK --interval=10s --timeout=5s CMD pg_isready

# pg_upgrade requires read and write access to the current directory.
WORKDIR /var/lib/postgresql

USER 70:70

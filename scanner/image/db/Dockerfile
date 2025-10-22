# Migration to ubi8-micro base image
# Scanner DB needs PostgreSQL server and many runtime dependencies

ARG RPMS_REGISTRY=registry.access.redhat.com
ARG RPMS_BASE_IMAGE=ubi8
ARG RPMS_BASE_TAG=latest

ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8-micro
ARG BASE_TAG=latest

# ============================================================================
# Stage 1: postgres_rpms - Download PostgreSQL RPMs
# ============================================================================
FROM ${RPMS_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} AS postgres_rpms

COPY scripts/download.sh /download.sh
RUN /download.sh

# ============================================================================
# Stage 2: dependency_builder - Install all runtime dependencies
# PostgreSQL DB needs many packages for runtime operation
# ============================================================================
FROM ${RPMS_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} AS dependency_builder

# If this is updated, be sure to update postgres_major in download.sh and the signature file.
ENV PG_MAJOR=15

# Install all required runtime dependencies to /out/
RUN dnf install \
    --installroot=/out/ \
    --releasever=8 \
    --setopt=install_weak_deps=0 \
    --nodocs \
    -y \
    shadow-utils \
    ca-certificates \
    glibc-langpack-en \
    glibc-locale-source \
    libicu \
    libxslt \
    lz4 \
    perl-libs \
    python3 \
    systemd-sysv \
    zstd \
    tzdata && \
    if [[ $(awk -F'=' '/VERSION_ID/{ gsub(/"/,""); print substr($2,1,1)}' /etc/os-release) -gt 8 ]]; then \
        dnf install --installroot=/out/ --releasever=8 --setopt=install_weak_deps=0 --nodocs -y uuid; \
    fi && \
    dnf --installroot=/out/ clean all && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum

# Copy PostgreSQL RPMs from postgres_rpms stage
COPY --from=postgres_rpms /rpms/postgres.rpm /rpms/postgres-libs.rpm /rpms/postgres-server.rpm /rpms/postgres-contrib.rpm /tmp/

# Import PostgreSQL GPG key and install RPMs to /out/
COPY signatures/PGDG-RPM-GPG-KEY-RHEL /tmp/
RUN rpm --root=/out/ --import /tmp/PGDG-RPM-GPG-KEY-RHEL && \
    rpm --root=/out/ -ivh --nodeps /tmp/postgres-libs.rpm /tmp/postgres-server.rpm /tmp/postgres.rpm /tmp/postgres-contrib.rpm && \
    rm -rf /tmp/*.rpm /tmp/PGDG-RPM-GPG-KEY-RHEL

# Create postgres user and group in /out/ with UID/GID 70
# Note: PostgreSQL RPMs may create these during installation, but we need specific UID/GID
# If they exist with wrong IDs, modify them; otherwise create them
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
    fi \
    "

# Setup locale in /out/
RUN chroot /out /bin/sh -c "localedef -f UTF-8 -i en_US en_US.UTF-8"

# Create required directories in /out/
RUN mkdir -p /out/docker-entrypoint-initdb.d

# ============================================================================
# Stage 3: FINAL - ubi8-micro runtime image
# NO package manager operations allowed in this stage!
# ============================================================================
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

# If this is updated, be sure to update postgres_major in download.sh and the signature file.
ENV PG_MAJOR=15
ENV PATH="$PATH:/usr/pgsql-$PG_MAJOR/bin/"
ENV LANG=en_US.utf8

# Copy all dependencies and PostgreSQL from builder
COPY --from=dependency_builder /out/ /

# Copy application files
# This will be ignored if empty in the init script.
COPY init-bundles/db-init.dump.zst /db-init.dump.zst
COPY scripts/docker-entrypoint.sh scripts/init-entrypoint.sh /usr/local/bin/

STOPSIGNAL SIGINT

# This is equivalent to postgres:postgres.
USER 70:70

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 5432
CMD ["postgres", "-c", "config_file=/etc/stackrox.d/config/postgresql.conf"]

ARG FINAL_STAGE_PATH="/mnt/final"

ARG RPMS_REGISTRY=registry.access.redhat.com
ARG RPMS_BASE_IMAGE=ubi8
ARG RPMS_BASE_TAG=8.7

ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8-minimal
ARG BASE_TAG=8.7

FROM ${RPMS_REGISTRY}/${RPMS_BASE_IMAGE}:${RPMS_BASE_TAG} AS ubi-base
FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG} AS final-base


FROM ubi-base AS postgres_rpms
ARG FINAL_STAGE_PATH
COPY --from=final-base / "$FINAL_STAGE_PATH"

COPY scripts/download.sh /download.sh
RUN /download.sh

COPY signatures/PGDG-RPM-GPG-KEY-RHEL /

RUN dnf -y --installroot="$FINAL_STAGE_PATH" upgrade -y --nobest && \
    # groupadd is in shadow-utils package that is not installed by default.
    dnf -y --installroot="$FINAL_STAGE_PATH" install -y shadow-utils && \
    rpm --import PGDG-RPM-GPG-KEY-RHEL && \
    dnf -y --installroot="$FINAL_STAGE_PATH" install -y \
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
        && \
    if [[ $(awk -F'=' '/VERSION_ID/{ gsub(/"/,""); print substr($2,1,1)}' /etc/os-release) -gt 8 ]]; then \
        dnf -y --installroot="$FINAL_STAGE_PATH" install -y uuid; \
    fi && \
    rpm --root $FINAL_STAGE_PATH -i /rpms/postgres-libs.rpm /rpms/postgres-server.rpm /rpms/postgres.rpm /rpms/postgres-contrib.rpm && \
    # Restore /usr/share/zoneinfo that's empty in ubi-minimal because postgres reads timezone data from it.
    # https://access.redhat.com/solutions/5616681
    dnf -y --installroot="$FINAL_STAGE_PATH" reinstall -y tzdata && \
    dnf -y --installroot="$FINAL_STAGE_PATH" clean all

FROM scratch

ARG FINAL_STAGE_PATH

COPY --from=postgres_rpms "$FINAL_STAGE_PATH" /

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

# This will be ignored if empty in the init script.
COPY init-bundles/db-init.dump.zst /db-init.dump.zst

COPY scripts/docker-entrypoint.sh scripts/init-entrypoint.sh /usr/local/bin/

RUN localedef -f UTF-8 -i en_US en_US.UTF-8 && \
    mkdir -p /var/lib/postgresql && \
    groupmod -g 70 postgres && \
    usermod -u 70 postgres -d /var/lib/postgresql && \
    chown -R postgres:postgres /var/lib/postgresql && \
    chown -R postgres:postgres /var/run/postgresql && \
    mkdir /docker-entrypoint-initdb.d

STOPSIGNAL SIGINT

# This is equivalent to postgres:postgres.
USER 70:70

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 5432
CMD ["postgres", "-c", "config_file=/etc/stackrox.d/config/postgresql.conf"]

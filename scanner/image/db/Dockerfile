ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8-minimal
ARG BASE_TAG=8.8

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

LABEL name="scanner-db" \
      vendor="StackRox" \
      maintainer="support@stackrox.com" \
      summary="Static vulnerability scanner database for the StackRox Security Platform" \
      description="This image supports static vulnerability scanning for the StackRox Security Platform."

# If this is updated, be sure to update the version of the postgres server installed below.
ENV PG_MAJOR=13
ENV PATH="$PATH:/usr/pgsql-$PG_MAJOR/bin/" \
    PGDATA="/var/lib/postgresql/data/pgdata"
ENV LANG=en_US.utf8

COPY scripts/docker-entrypoint.sh /usr/local/bin/

ARG POSTGRESQL_ARCH=x86_64

RUN curl -sSLf https://download.postgresql.org/pub/repos/yum/RPM-GPG-KEY-PGDG-${PG_MAJOR} -o /tmp/pg_repo.key && \
    rpm --import /tmp/pg_repo.key && \
    curl -sSLf https://download.postgresql.org/pub/repos/yum/reporpms/EL-8-${POSTGRESQL_ARCH}/pgdg-redhat-repo-latest.noarch.rpm -o /tmp/pg_repo.rpm && \
    rpm -i /tmp/pg_repo.rpm && \
    # Skip repo check for ARM as it's used only for development purposes and package has wrong signature.
    if [[ "$POSTGRESQL_ARCH" == "aarch64" ]]; then sed -i 's/repo_gpgcheck = 1/repo_gpgcheck = 0/g' /etc/yum.repos.d/pgdg-redhat-all.repo; fi && \
    microdnf upgrade -y && \
    microdnf install -y shadow-utils && \
    groupadd -g 70 postgres && \
    adduser postgres -u 70 -g 70 -d /var/lib/postgresql -s /bin/sh && \
    microdnf install -y \
        ca-certificates libicu systemd-sysv glibc-locale-source glibc-langpack-en \
        postgresql13-server && \
    # The removal of /usr/share/zoneinfo from UBI minimal images is intentional.
    # After building the image, the image is reduced in size as much as possible,
    # and the /usr/share/zoneinfo directory is purged as it saves space
    # in the final distribution of the image.
    # https://access.redhat.com/solutions/5616681
    microdnf reinstall tzdata && \
    microdnf clean all && \
    rpm -e --nodeps $(rpm -qa 'pgdg-redhat-repo*') && \
    # (Optional) Remove line below to keep package management utilities
    rpm -e --nodeps $(rpm -qa shadow-utils curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum  /tmp/pg_repo.rpm /tmp/pg_repo.key && \
    localedef -f UTF-8 -i en_US en_US.UTF-8 && \
    chown postgres:postgres /usr/local/bin/docker-entrypoint.sh && \
    chmod +x /usr/local/bin/docker-entrypoint.sh && \
    mkdir /docker-entrypoint-initdb.d

STOPSIGNAL SIGINT

# This is equivalent to postgres:postgres.
USER 70:70

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 5432
CMD ["postgres", "-c", "config_file=/etc/stackrox.d/config/postgresql.conf"]

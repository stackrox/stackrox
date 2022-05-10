ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8/ubi
ARG BASE_TAG=8.5

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG} AS extracted_bundle

ADD bundle.tar.gz /bundle

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

LABEL name="central-db" \
      vendor="StackRox" \
      maintainer="support@stackrox.com" \
      summary="Stackrox Central Database for the StackRox Kubernetes Security Platform" \
      description="This image provides Database services to Stackrox Central in the StackRox Kubernetes Security Platform."

ARG ROX_IMAGE_FLAVOR
ENV ROX_IMAGE_FLAVOR=${ROX_IMAGE_FLAVOR}

ENV PG_MAJOR=14
ENV PATH="$PATH:/usr/pgsql-$PG_MAJOR/bin/"

COPY scripts/docker-entrypoint.sh /usr/local/bin/
COPY --from=extracted_bundle /bundle/etc/postgresql.conf /bundle/etc/pg_hba.conf /etc/

RUN groupadd -g 70 postgres && \
    adduser postgres -u 70 -g 70 -d /var/lib/postgresql -s /bin/sh && \
    dnf upgrade -y --nobest && \
    dnf install -y \
        ca-certificates libicu systemd-sysv \
        glibc-locale-source glibc-langpack-en \
        https://download.postgresql.org/pub/repos/yum/reporpms/EL-8-x86_64/pgdg-redhat-repo-latest.noarch.rpm && \
    dnf install -y postgresql14-server && \
    dnf clean all && \
    rpm -e --nodeps $(rpm -qa 'pgdg-redhat-repo*') && \
    rpm -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum && \
    localedef -f UTF-8 -i en_US en_US.UTF-8 && \
    chown postgres:postgres /usr/local/bin/docker-entrypoint.sh && \
    chmod +x /usr/local/bin/docker-entrypoint.sh && \
    mkdir /docker-entrypoint-initdb.d

USER postgres:postgres

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 5432
CMD ["postgres", "-c", "config_file=/etc/postgresql.conf"]

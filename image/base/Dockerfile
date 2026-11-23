# Pre-built base image for StackRox main image.
# Built WEEKLY (or on Dockerfile change) by .github/workflows/base-image.yaml.
# Referenced by the main Dockerfile as a pinned FROM.
#
# This separates the SLOW-CHANGING layers (OS packages, RPM installs)
# from the FAST-CHANGING layers (binaries, UI). CI builds reference this
# image and never run dnf, eliminating:
#   - 10-15s of package mirror round-trips per build
#   - Network dependency on Red Hat/pgdg mirrors (outage class)
#   - Base layer verification time on cached builds
#
# Mirrors master's runtime-base construction exactly (downloads +
# package_installer stages): PostgreSQL client tools (createdb, pg_dump,
# ...) are REQUIRED at runtime — central's migrator shells out to
# createdb — and ubi-micro has no dnf, so packages install via
# --installroot in the builder stage and are layered onto micro.

ARG PG_VERSION=16

FROM registry.access.redhat.com/ubi9:latest AS downloads

ARG PG_VERSION
COPY image/rhel/signatures/RPM-GPG-KEY-CentOS-Official /tmp/
COPY image/rhel/download.sh /download.sh
ENV DEBUG_BUILD=no
RUN /download.sh && rm -f /download.sh

FROM registry.access.redhat.com/ubi9:latest AS package_installer

ARG PG_VERSION
COPY --from=downloads /output/rpms/ /tmp/
COPY image/rhel/signatures/RPM-GPG-KEY-CentOS-Official /tmp/

RUN rpm --import /tmp/RPM-GPG-KEY-CentOS-Official && \
    dnf install -y \
        --installroot=/out/ \
        --releasever=9 \
        --setopt=install_weak_deps=0 \
        --nodocs \
        findutils \
        ca-certificates \
        gzip \
        less \
        tar \
        /tmp/postgres-libs.rpm \
        /tmp/postgres.rpm && \
    dnf clean all --installroot=/out/ && \
    rm -rf /out/var/cache/dnf /out/var/cache/yum

FROM registry.access.redhat.com/ubi9/ubi-micro:latest

ARG PG_VERSION

COPY --from=package_installer /out/ /

# Execute the clients in the final runtime to check their version and
# libraries (mirrors master's runtime-base guard).
RUN set -eu; \
    for client in pg_dump pg_restore createdb; do \
        version="$("$client" --version)"; \
        echo "$version"; \
        case "$version" in \
            "$client (PostgreSQL) ${PG_VERSION}."*) ;; \
            *) echo "Expected PostgreSQL ${PG_VERSION} client: $client" >&2; exit 1 ;; \
        esac; \
    done

# Scratch dirs that central expects at runtime
RUN mkdir -p /stackrox/bin /stackrox/static-data /ui /var/lib/stackrox && \
    chown -R 4000:4000 /stackrox /ui /var/lib/stackrox

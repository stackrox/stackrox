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

FROM registry.access.redhat.com/ubi9:latest@sha256:87f66688da4e206d5a1877088626c542288a5ccd97898cf363b588d57817e6be

COPY image/rhel/signatures/RPM-GPG-KEY-CentOS-Official /tmp/
COPY image/rhel/download.sh /download.sh
ENV DEBUG_BUILD=no
RUN /download.sh && rm -f /download.sh /tmp/RPM-GPG-KEY-CentOS-Official

# Scratch dirs that central expects at runtime
RUN mkdir -p /stackrox/bin /stackrox/static-data /ui /var/lib/stackrox && \
    chown -R 4000:4000 /stackrox /ui /var/lib/stackrox

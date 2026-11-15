# Minimal product image — the content-addressed build target.
# FROM pre-built base (pinned, weekly) + COPY product artifacts.
# No dnf. No RPM installs. No multi-arch roxctl copies. No ifdef CI.
# Pure layer assembly from cached content.

FROM quay.io/stackrox-io/ubi-base:latest

COPY --link bin/central /stackrox/central
COPY --link bin/config-controller /stackrox/bin/config-controller
COPY --link bin/central-worker /stackrox/bin/central-worker
COPY --link ui/build/ /ui/

EXPOSE 8443
USER 4000:4000
ENTRYPOINT ["/stackrox/central"]

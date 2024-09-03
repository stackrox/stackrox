ARG BASE_REGISTRY=registry.access.redhat.com
ARG BASE_IMAGE=ubi8-minimal
ARG BASE_TAG=latest

FROM ${BASE_REGISTRY}/${BASE_IMAGE}:${BASE_TAG}

ARG LABEL_VERSION
ARG LABEL_RELEASE
ARG QUAY_TAG_EXPIRATION

LABEL name="definitions" \
      vendor="StackRox" \
      maintainer="https://stackrox.io/" \
      summary="Vulnerability definitions for the StackRox Security Platform" \
      description="This image contains vulnerability definitions used by StackRox Scanner V4." \
      version="${LABEL_VERSION}" \
      release="${LABEL_RELEASE}" \
      quay.expires-after="${QUAY_TAG_EXPIRATION}"

COPY vulns /definitions

# This is equivalent to nobody:nobody.
USER 65534:65534

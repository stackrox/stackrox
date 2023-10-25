# openshift-golang-builder is the only way to get more recent Go version than the official ubi8/go-toolset provides.
# See https://issues.redhat.com/browse/RHELPLAN-167618
# Using that has few known issues:
# - https://issues.redhat.com/browse/RHTAPBUGS-864 - deprecated-base-image-check behaves incorrectly.
# - https://issues.redhat.com/browse/RHTAPBUGS-865 - openshift-golang-builder is not considered to be a valid base image.
#
# TODO(ROX-20312): we can't pin image tag or digest because currently there's no mechanism to auto-update that.
# We're targeting a floating tag here which should be reasonably safe to do as both RHEL major 8 and Go major.minor 1.20 should provide enough stability.
FROM registry.access.redhat.com/ubi8/go-toolset:1.19 as builder

WORKDIR /go/src/github.com/stackrox/rox/app

RUN git config --global --add safe.directory /go/src/github.com/stackrox/rox/app && \
    # TODO(ROX-20233): Fetch git tags outside of Dockerfile
    git tag && \
    mkdir -p image/bin

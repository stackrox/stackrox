FROM registry.access.redhat.com/ubi9:latest as builder-runner
RUN dnf install -y --nodocs --noplugins --refresh --best make git golang findutils python3 jq

# Use a new stage to enable caching of the package installations for local development
FROM builder-runner as builder

COPY . /stackrox
WORKDIR /stackrox/operator

ARG MAIN_IMAGE_TAG
ENV VERSION=$MAIN_IMAGE_TAG
ENV ROX_PRODUCT_BRANDING=RHACS_BRANDING
RUN make bundle-post-process

FROM scratch

# Core bundle labels.
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=rhacs-operator
LABEL operators.operatorframework.io.bundle.channels.v1=latest
LABEL operators.operatorframework.io.bundle.channel.default.v1=latest
LABEL operators.operatorframework.io.metrics.builder=operator-sdk-unknown
LABEL operators.operatorframework.io.metrics.mediatype.v1=metrics+v1
LABEL operators.operatorframework.io.metrics.project_layout=go.kubebuilder.io/v3

# Labels for testing.
LABEL operators.operatorframework.io.test.mediatype.v1=scorecard+v1
LABEL operators.operatorframework.io.test.config.v1=tests/scorecard/

# Labels for operator certification https://redhat-connect.gitbook.io/certified-operator-guide/ocp-deployment/operator-metadata/bundle-directory
LABEL com.redhat.delivery.operator.bundle=true

# This sets the earliest version of OCP where our operator build would show up in the official Red Hat operator catalog.
# vX means "X or later": https://redhat-connect.gitbook.io/certified-operator-guide/ocp-deployment/operator-metadata/bundle-directory/managing-openshift-versions
#
# The version here should stay the lowest not yet EOL so that downstream CVP tests don't fail.
# See EOL schedule: https://docs.engineering.redhat.com/display/SP/Shipping+Operators+to+EOL+OCP+versions
#
# See https://docs.engineering.redhat.com/display/StackRox/Add+support+for+new+OpenShift+version#AddsupportfornewOpenShiftversion-RemovesupportforOpenShiftversionwentEOL
# for info when to adjust this version.
LABEL com.redhat.openshift.versions="v4.12"

# Use post-processed files (instead of the original ones).
COPY --from=builder /stackrox/operator/build/bundle/manifests /manifests/
COPY --from=builder /stackrox/operator/build/bundle/metadata /metadata/
COPY --from=builder /stackrox/operator/build/bundle/tests/scorecard /tests/scorecard/

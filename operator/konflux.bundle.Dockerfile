FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_8_golang_1.25@sha256:527782f4a0270f786192281f68d0374f4a21b3ab759643eee4bfcafb6f539468 AS builder

COPY . /stackrox
WORKDIR /stackrox/operator

ARG OPERATOR_IMAGE_TAG
RUN echo "Checking required OPERATOR_IMAGE_TAG"; [[ "${OPERATOR_IMAGE_TAG}" != "" ]]

ARG OPERATOR_IMAGE_REF
RUN echo "Checking required OPERATOR_IMAGE_REF"; [[ "${OPERATOR_IMAGE_REF}" != "" ]]

ARG RELATED_IMAGE_MAIN
ENV RELATED_IMAGE_MAIN=$RELATED_IMAGE_MAIN
RUN echo "Checking required RELATED_IMAGE_MAIN"; [[ "${RELATED_IMAGE_MAIN}" != "" ]]

ARG RELATED_IMAGE_SCANNER
ENV RELATED_IMAGE_SCANNER=$RELATED_IMAGE_SCANNER
RUN echo "Checking required RELATED_IMAGE_SCANNER"; [[ "${RELATED_IMAGE_SCANNER}" != "" ]]

ARG RELATED_IMAGE_SCANNER_DB
ENV RELATED_IMAGE_SCANNER_DB=$RELATED_IMAGE_SCANNER_DB
RUN echo "Checking required RELATED_IMAGE_SCANNER_DB"; [[ "${RELATED_IMAGE_SCANNER_DB}" != "" ]]

ARG RELATED_IMAGE_SCANNER_SLIM
ENV RELATED_IMAGE_SCANNER_SLIM=$RELATED_IMAGE_SCANNER_SLIM
RUN echo "Checking required RELATED_IMAGE_SCANNER_SLIM"; [[ "${RELATED_IMAGE_SCANNER_SLIM}" != "" ]]

ARG RELATED_IMAGE_SCANNER_DB_SLIM
ENV RELATED_IMAGE_SCANNER_DB_SLIM=$RELATED_IMAGE_SCANNER_DB_SLIM
RUN echo "Checking required RELATED_IMAGE_SCANNER_DB_SLIM"; [[ "${RELATED_IMAGE_SCANNER_DB_SLIM}" != "" ]]

ARG RELATED_IMAGE_SCANNER_V4
ENV RELATED_IMAGE_SCANNER_V4=$RELATED_IMAGE_SCANNER_V4
RUN echo "Checking required RELATED_IMAGE_SCANNER_V4"; [[ "${RELATED_IMAGE_SCANNER_V4}" != "" ]]

ARG RELATED_IMAGE_SCANNER_V4_DB
ENV RELATED_IMAGE_SCANNER_V4_DB=$RELATED_IMAGE_SCANNER_V4_DB
RUN echo "Checking required RELATED_IMAGE_SCANNER_V4_DB"; [[ "${RELATED_IMAGE_SCANNER_V4_DB}" != "" ]]

ARG RELATED_IMAGE_COLLECTOR
ENV RELATED_IMAGE_COLLECTOR=$RELATED_IMAGE_COLLECTOR
RUN echo "Checking required RELATED_IMAGE_COLLECTOR"; [[ "${RELATED_IMAGE_COLLECTOR}" != "" ]]

ARG RELATED_IMAGE_FACT
ENV RELATED_IMAGE_FACT=$RELATED_IMAGE_FACT
RUN echo "Checking required RELATED_IMAGE_FACT"; [[ "${RELATED_IMAGE_FACT}" != "" ]]

ARG RELATED_IMAGE_ROXCTL
ENV RELATED_IMAGE_ROXCTL=$RELATED_IMAGE_ROXCTL
RUN echo "Checking required RELATED_IMAGE_ROXCTL"; [[ "${RELATED_IMAGE_ROXCTL}" != "" ]]

ARG RELATED_IMAGE_CENTRAL_DB
ENV RELATED_IMAGE_CENTRAL_DB=$RELATED_IMAGE_CENTRAL_DB
RUN echo "Checking required RELATED_IMAGE_CENTRAL_DB"; [[ "${RELATED_IMAGE_CENTRAL_DB}" != "" ]]

# Build csv-patcher and fix-spec-descriptors from source
RUN cd /stackrox/operator/cmd/csv-patcher && go build -o /usr/local/bin/csv-patcher .
RUN cd /stackrox/operator/cmd/fix-spec-descriptors && go build -o /usr/local/bin/fix-spec-descriptors .

# Generate initial bundle
RUN /stackrox/operator/bundle_helpers/generate-bundle.sh \
      --use-version="${OPERATOR_IMAGE_TAG}" \
      --first-version=4.0.0 \
      --operator-image="${OPERATOR_IMAGE_REF}"

# Patch the CSV with related images for Konflux
RUN /usr/local/bin/csv-patcher \
      --csv-file=build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
      --operator-version="${OPERATOR_IMAGE_TAG}" \
      --operator-image="${OPERATOR_IMAGE_REF}" \
      --main-image="${RELATED_IMAGE_MAIN}" \
      --scanner-image="${RELATED_IMAGE_SCANNER}" \
      --scanner-db-image="${RELATED_IMAGE_SCANNER_DB}" \
      --scanner-slim-image="${RELATED_IMAGE_SCANNER_SLIM}" \
      --scanner-db-slim-image="${RELATED_IMAGE_SCANNER_DB_SLIM}" \
      --scanner-v4-image="${RELATED_IMAGE_SCANNER_V4}" \
      --scanner-v4-db-image="${RELATED_IMAGE_SCANNER_V4_DB}" \
      --collector-image="${RELATED_IMAGE_COLLECTOR}" \
      --roxctl-image="${RELATED_IMAGE_ROXCTL}" \
      --central-db-image="${RELATED_IMAGE_CENTRAL_DB}" \
      --output-file=build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml

# Fix spec descriptors
RUN /usr/local/bin/fix-spec-descriptors \
      --csv-file=build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
      --output-file=build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml

FROM scratch

ARG OPERATOR_IMAGE_TAG

# Enterprise Contract labels.
LABEL com.redhat.component="rhacs-operator-bundle-container"
LABEL com.redhat.license_terms="https://www.redhat.com/agreements"
LABEL description="Operator Bundle Image for Red Hat Advanced Cluster Security for Kubernetes"
LABEL distribution-scope="public"
LABEL io.k8s.description="Operator Bundle Image for Red Hat Advanced Cluster Security for Kubernetes"
LABEL io.k8s.display-name="operator-bundle"
LABEL io.openshift.tags="rhacs,operator-bundle,stackrox"
LABEL maintainer="Red Hat, Inc."
LABEL name="advanced-cluster-security/rhacs-operator-bundle"
# Custom Snapshot creation in `operator-bundle-pipeline` depends on source-location label to be set correctly.
LABEL source-location="https://github.com/stackrox/stackrox"
LABEL summary="Operator Bundle Image for Red Hat Advanced Cluster Security for Kubernetes"
LABEL url="https://catalog.redhat.com/software/container-stacks/detail/60eefc88ee05ae7c5b8f041c"
LABEL vendor="Red Hat, Inc."
# We must set version label to prevent inheriting value set in the base stage.
LABEL version="${OPERATOR_IMAGE_TAG}"
# Release label is required by EC although has no practical semantics.
# We also set it to not inherit one from a base stage in case it's RHEL or UBI.
LABEL release="1"

# Core bundle labels.
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=rhacs-operator
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

COPY LICENSE /licenses/LICENSE

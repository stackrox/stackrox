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
# Note: vX means "X or later": https://redhat-connect.gitbook.io/certified-operator-guide/ocp-deployment/operator-metadata/bundle-directory/managing-openshift-versions
LABEL com.redhat.openshift.versions="v4.11"
LABEL com.redhat.delivery.operator.bundle=true

# Use post-processed files (instead of the original ones).
COPY build/bundle/manifests /manifests/
COPY build/bundle/metadata /metadata/
COPY build/bundle/tests/scorecard /tests/scorecard/

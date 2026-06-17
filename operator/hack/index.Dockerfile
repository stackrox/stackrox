# Keep OPM version in sync with operator-index repository OPM:
#   https://github.com/stackrox/operator-index/blob/master/Makefile#L3
ARG OPM_BASE_IMAGE=quay.io/operator-framework/opm:v1.48.0

FROM ${OPM_BASE_IMAGE}

ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs", "--cache-dir=/tmp/cache"]

ADD rhacs-operator-index /configs
LABEL operators.operatorframework.io.index.configs.v1=/configs

# Build the cache during image build for multi-platform compatibility.
# Without this, OLM would try to build the cache at runtime by pulling bundle
# images, which fails on non-amd64 platforms when bundles are amd64-only.
RUN ["/bin/opm", "serve", "/configs", "--cache-dir=/tmp/cache", "--cache-only"]

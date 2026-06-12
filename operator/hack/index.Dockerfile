# Multi-platform operator index Dockerfile
# This replaces the OPM-generated Dockerfile to support multi-platform builds.
#
# The key difference from `opm generate dockerfile` output is the cache-building
# step, which ensures ARM/ppc64le/s390x clusters can use the index without
# needing to pull single-platform (amd64) bundles at runtime.

ARG OPM_BASE_IMAGE=quay.io/operator-framework/opm:v1.48.0

FROM ${OPM_BASE_IMAGE}

# Configure the entrypoint and command
ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs", "--cache-dir=/tmp/cache"]

# Copy declarative config root into image at /configs
ADD rhacs-operator-index /configs

# Set DC-specific label for the location of the DC root directory in the image
LABEL operators.operatorframework.io.index.configs.v1=/configs

# Build the cache during image build for multi-platform compatibility.
# Without this, OLM would try to build the cache at runtime by pulling bundle
# images, which fails on non-amd64 platforms when bundles are amd64-only.
RUN ["/bin/opm", "serve", "/configs", "--cache-dir=/tmp/cache", "--cache-only"]

#!/bin/bash
# Builds operator index image.

set -eou pipefail

# Global script variables
OPM_VERSION="1.21.0"
YQ_VERSION="4.24.2"

function usage() {
  echo "
Usage:
  build-index-image.sh MANDATORY [OPTION]

MANDATORY:
  --base-index-tag     The base index image tag. Example: quay.io/stackrox-io/stackrox-operator-index:v1.0.0
  --index-tag          The new index image tag. Example: quay.io/stackrox-io/stackrox-operator-index:v1.1.0
  --bundle-tag         The bundle image tag that should be appended to base index. Example: quay.io/stackrox-io/stackrox-operator-bundle:v1.1.0
  --replaced-version   Version that the bundle replaces. Example: v1.0.0

OPTION:
  --base-dir           Working directory for the script. Default: '.'
  --clean-output-dir   Delete '{base-dir}/build/index' directory.
  --use-http           Use plain HTTP for container image registries.
  --skip-build         Skip the actual \"docker build\" command.
  --skip-tls-verify    Skip TLS certificate verification for container image registries while pulling bundles
" >&2
}

function usage_exit() {
  usage
  exit 1
}

# Script argument variables
BASE_INDEX_TAG=""
INDEX_TAG=""
BUNDLE_TAG=""
REPLACED_VERSION=""
BASE_DIR="."
RUN_BUILD=1

# Helpful for local development and testing
CLEAN_OUTPUT_DIR=""
USE_HTTP=""
SKIP_TLS_VERIFY=""

function read_arguments() {
    while [[ -n "${1:-}" ]]; do
        case "${1}" in
            "--base-index-tag")
                BASE_INDEX_TAG="${2}";shift;;
            "--index-tag")
                INDEX_TAG="${2}";shift;;
            "--bundle-tag")
                BUNDLE_TAG="${2}";shift;;
            "--replaced-version")
                REPLACED_VERSION="${2}";shift;;
            "--base-dir")
                BASE_DIR="${2}";shift;;
            "--clean-output-dir")
                CLEAN_OUTPUT_DIR="true";;
            "--use-http")
                USE_HTTP="--use-http";;
            "--skip-tls-verify")
                SKIP_TLS_VERIFY="--skip-tls-verify";;
            "--skip-build")
                RUN_BUILD=0;;
            *)
                echo "Error: Unknown parameter: ${1}" >&2
                usage_exit
        esac

        if ! shift; then
            echo 'Error: Missing parameter argument.' >&2
            usage_exit
        fi
    done
}

function validate_arguments() {
  [[ "${BASE_INDEX_TAG}" = "" ]] && echo "Error: Base index tag is required." >&2 && usage_exit
  [[ "${INDEX_TAG}" = "" ]] && echo "Error: Index tag is required." >&2 && usage_exit
  [[ "${BUNDLE_TAG}" = "" ]] && echo "Error: Bundle tag is required." >&2 && usage_exit
  [[ "${REPLACED_VERSION}" = "" ]] && echo "Error: Replaced version is required." >&2 && usage_exit

  return 0
}

SCRIPT_DIR=$(dirname "${BASH_SOURCE[0]}")

OPM="opm"
function fetch_opm() {
  local -r os_name=$(uname | tr '[:upper:]' '[:lower:]') || true
  local -r arch=$(go env GOARCH) || true

  OPM="${BASE_DIR}/bin/opm-${OPM_VERSION}"
  "${SCRIPT_DIR}/get-github-release.sh" --to "${OPM}" --from "https://github.com/operator-framework/operator-registry/releases/download/v${OPM_VERSION}/${os_name}-${arch}-opm"
}

YQ="yq"
function fetch_yq() {
  local -r os_name=$(uname | tr '[:upper:]' '[:lower:]') || true
  local -r arch=$(go env GOARCH) || true

  YQ="${BASE_DIR}/bin/yq-${YQ_VERSION}"
  "${SCRIPT_DIR}/get-github-release.sh" --to "${YQ}" --from "https://github.com/mikefarah/yq/releases/download/v${YQ_VERSION}/yq_${os_name}_${arch}"
}

# Script body
read_arguments "$@"
validate_arguments
fetch_opm
fetch_yq

if [[ "${CLEAN_OUTPUT_DIR}" = "true" ]]; then
  rm -rf "${BASE_DIR}/build/index"
fi

if [[ "${REPLACED_VERSION}" = v* ]]; then
  REPLACED_VERSION="${REPLACED_VERSION:1}"
fi

echo "Detected that bundle ${BUNDLE_TAG} updates (replaces) version ${REPLACED_VERSION}. Will use index image ${BASE_INDEX_TAG} as the base for the current one." >&2

# Exports for docker build and opm in case it builds the image
export DOCKER_BUILDKIT=1
export BUILDKIT_PROGRESS="plain"

BUILD_INDEX_DIR="${BASE_DIR}/build/index/rhacs-operator-index"
mkdir -p "${BUILD_INDEX_DIR}"

# With "--binary-image", we are setting the exact base image version. By default, "latest" would be used.
"${OPM}" generate dockerfile --binary-image "quay.io/operator-framework/opm:v${OPM_VERSION}" "${BUILD_INDEX_DIR}"
"${OPM}" render "${BASE_INDEX_TAG}" --output=yaml ${USE_HTTP} ${SKIP_TLS_VERIFY} > "${BUILD_INDEX_DIR}/index.yaml"

BUNDLE_VERSION="${BUNDLE_TAG##*:v}"
YQ_FILTER_CHANNEL_DOCUMENT='.schema=="olm.channel" and .name=="latest"'
YQ_NEW_BUNDLE_ENTRY=$(cat <<EOF
{
    "name": "rhacs-operator.v${BUNDLE_VERSION}",
    "replaces": "rhacs-operator.v${REPLACED_VERSION}",
    "skipRange": ">= ${REPLACED_VERSION} < ${BUNDLE_VERSION}"
}
EOF
)

"${YQ}" --inplace --prettyPrint "with(select(${YQ_FILTER_CHANNEL_DOCUMENT}); .entries += ${YQ_NEW_BUNDLE_ENTRY})" "${BUILD_INDEX_DIR}/index.yaml"
"${OPM}" render "${BUNDLE_TAG}" --output=yaml ${USE_HTTP} ${SKIP_TLS_VERIFY} >> "${BUILD_INDEX_DIR}/index.yaml"
"${OPM}" validate "${BUILD_INDEX_DIR}"

if (( RUN_BUILD )); then
  docker build --quiet --file "${BUILD_INDEX_DIR}.Dockerfile" --tag "${INDEX_TAG}" "${BUILD_INDEX_DIR}/.."

  echo "Index image ${INDEX_TAG} is successfully created."
else
  echo "Skipping 'docker build' of ${BUILD_INDEX_DIR} as requested."
fi

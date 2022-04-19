#!/bin/bash
# Builds operator index image.

set -e -o pipefail

# Load utils
SCRIPT_DIR=$(dirname "${BASH_SOURCE[0]}")
# shellcheck source=SCRIPTDIR/utils.sh
source "${SCRIPT_DIR}/utils.sh"

# Global script variables
OPM_VERSION="1.21.0"
YQ_VERSION="4.24.2"

function usage() {
  echo "
Usage:
  make-index-build-step.sh [options]

Options:
  --base-dir           Working directory for the script. Default: '.'
  --base-index-tag     The base index image tag. Example: docker.io/stackrox/stackrox-operator-index:v1.0.0
  --index-tag          The new index image tag. Example: docker.io/stackrox/stackrox-operator-index:v1.1.0
  --bundle-tag         The bundle image tag that should be appended to base index. Example: docker.io/stackrox/stackrox-operator-bundle:v1.1.0
  --replaced-version   Version that the bundle replaces. Example: v1.0.0
  --clean-output-dir   Delete build output directory: '{base-dir}/build/index'.
  --use-http           Use plain HTTP for container image registries.
" 1>&2
}

function usage_exit() {
  usage
  exit 1
}

# Script argument variables
BASE_DIR="."
BASE_INDEX_TAG=""
REPLACED_VERSION=""
BUNDLE_TAG=""
INDEX_TAG=""

# Helpful for local development and testing
CLEAN_OUTPUT_DIR=""
USE_HTTP=""

function read_arguments() {
    while [[ "${1}" ]]; do
        case "${1}" in
            "--base-dir")
                BASE_DIR="${2}";shift;;
            "--base-index-tag")
                BASE_INDEX_TAG="${2}";shift;;
            "--replaced-version")
                REPLACED_VERSION="${2}";shift;;
            "--bundle-tag")
                BUNDLE_TAG="${2}";shift;;
            "--index-tag")
                INDEX_TAG="${2}";shift;;
            "--use-http")
                USE_HTTP="--use-http";;
            "--clean-output-dir")
                CLEAN_OUTPUT_DIR="true";;
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
  [[ "${BASE_INDEX_TAG}" = "" ]] && echo "Error: Base index tag is required." && usage_exit
  [[ "${REPLACED_VERSION}" = "" ]] && echo "Error: Replaced version is required." && usage_exit
  [[ "${BUNDLE_TAG}" = "" ]] && echo "Error: Bundle tag is required." && usage_exit
  [[ "${INDEX_TAG}" = "" ]] && echo "Error: Index tag is required." && usage_exit

  return 0
}

OPM="opm"
function fetch_opm() {
  local -r os_name=$(uname | tr '[:upper:]' '[:lower:]')

  OPM="${BASE_DIR}/bin/opm-${OPM_VERSION}"
  get_github_release --to "${OPM}" --from "https://github.com/operator-framework/operator-registry/releases/download/v${OPM_VERSION}/${os_name}-$(go env GOARCH)-opm"
}

YQ="yq"
function fetch_yq() {
  local -r os_name=$(uname | tr '[:upper:]' '[:lower:]')

  YQ="${BASE_DIR}/bin/yq-${YQ_VERSION}"
  get_github_release --to "${YQ}" --from "https://github.com/mikefarah/yq/releases/download/v${YQ_VERSION}/yq_${os_name}_$(go env GOARCH)"
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
"${OPM}" render "${BASE_INDEX_TAG}" --output=yaml ${USE_HTTP} > "${BUILD_INDEX_DIR}/index.yaml"

BUNDLE_VERSION=$(echo "${BUNDLE_TAG}" | awk -F: '{print $NF}')
BUNDLE_VERSION="${BUNDLE_VERSION:1}"
"${YQ}" --inplace --prettyPrint "with(select(.schema==\"olm.channel\" and .name==\"latest\"); .entries += {\"name\":\"rhacs-operator.v${BUNDLE_VERSION}\",\"replaces\":\"rhacs-operator.v${REPLACED_VERSION}\",\"skipRange\":\">= ${REPLACED_VERSION} < ${BUNDLE_VERSION}\"})" "${BUILD_INDEX_DIR}/index.yaml"
"${OPM}" render "${BUNDLE_TAG}" --output=yaml ${USE_HTTP} >> "${BUILD_INDEX_DIR}/index.yaml"
"${OPM}" validate "${BUILD_INDEX_DIR}"
docker build --quiet --file "${BUILD_INDEX_DIR}.Dockerfile" --tag "${INDEX_TAG}" "${BUILD_INDEX_DIR}/.."

echo "Index image ${INDEX_TAG} is successfully created."

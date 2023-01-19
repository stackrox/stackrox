#!/bin/bash
set -euo pipefail

if [[ "$#" -lt 4 ]]; then
    echo >&2 "missing args. usage: $0 <class> <description> <failure_message> <command> [ args ]"
    exit 1
fi
declare -r class="$1"; shift
declare -r description="$1"; shift
declare -r failure_message="$1"; shift

ROOT_DIR="$(dirname "${BASH_SOURCE[0]}")/../.."
readonly ROOT_DIR

if "$@"; then
    "${ROOT_DIR}/scripts/ci/lib.sh" "save_junit_success" "${class}" "${description}"
else
    declare -r ret_code="$?"
    "${ROOT_DIR}/scripts/ci/lib.sh" "save_junit_failure" "${class}" "${description}" "${failure_message}"
    exit ${ret_code}
fi

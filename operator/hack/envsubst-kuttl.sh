#!/bin/bash
# This script is used by some assert files in the test directory.
# It replaces $NAMESPACE in the file supplied as the first argument,
# and then runs kuttl with subsequent arguments on the result.
set -euo pipefail
input_file="$1"
shift
intermediate_file=$(mktemp)
trap 'rm -f ${intermediate_file}' EXIT
env - PATH="${PATH}" NAMESPACE="${NAMESPACE}" envsubst < "${input_file}" > "${intermediate_file}"
${KUTTL:-kubectl-kuttl} "$@" "${intermediate_file}"

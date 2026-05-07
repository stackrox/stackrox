#!/usr/bin/env bash

# This script validates that DEBUG_BUILD is not enabled for tagged builds.
# Binary verification removed to support reproducible builds with direct registry push.

set -euo pipefail

errecho() {
  echo >&2 -e "$@"
}

if [[ -n "${BUILD_TAG}" && "${DEBUG_BUILD}" == "yes" ]]; then
  errecho "BUILD_TAG environment variable is set. DEBUG_BUILD-s are not supported with tagged, e.g. release or nightly, builds."
  errecho "Failing the build. Please make sure DEBUG_BUILD variable is not manually overridden to \"yes\"."
  exit 2
fi

echo "Debug build validation passed (DEBUG_BUILD=${DEBUG_BUILD}, BUILD_TAG=${BUILD_TAG:-<unset>})"

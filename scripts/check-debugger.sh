#!/usr/bin/env bash

# This script is to prevent release image from having dlv debugger in it.
# This script must be called during CI.
# This script must be called from the Makefile which sets necessary environment variables.

set -euo pipefail

errecho() {
  echo >&2 -e "$@"
}

if [[ -n "${BUILD_TAG}" && "${DEBUG_BUILD}" == "yes" ]]; then
  errecho "BUILD_TAG environment variable is set. DEBUG_BUILD-s are not supported with tagged, e.g. release or nightly, builds."
  errecho "Failing the build. Please make sure DEBUG_BUILD variable is not manually overridden to \"yes\"."
  exit 2
fi

# This searches for a file in the image without running the container.
container=$(docker create stackrox/main:"${TAG}")
docker export "${container}" | tar t | grep 'bin/dlv$' && found_dlv="yes" || found_dlv="no"
docker rm "${container}" &>/dev/null

if [[ "${found_dlv}" != "${DEBUG_BUILD}" ]]; then
  if [[ "${DEBUG_BUILD}" == "yes" ]]; then
    errecho "Warning: dlv debugger not found in the resulting image"
  else
    errecho "Detected dlv debugger binary in the resulting image while DEBUG_BUILD is off. There must be a problem with build scripts."
    errecho "Failing the build because vending debugger with the application is insecure."
    exit 3
  fi
fi

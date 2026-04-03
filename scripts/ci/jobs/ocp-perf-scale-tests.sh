#!/usr/bin/env bash

# Wait for PR images to be available for performance/scale testing.
# This script ONLY polls for images and exports the tag.
# Installation is handled by stackrox-install-helm in openshift/release.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"../../.. && pwd)"
# shellcheck source=../lib.sh
source "$ROOT/scripts/ci/lib.sh"

info "Waiting for PR images for performance/scale testing"

# Calculate the image tag
ACS_VERSION_TAG="$(make --quiet --no-print-directory tag)"
info "Target image tag: ${ACS_VERSION_TAG}"

# Wait for PR images to be built and available
POLL_TIMEOUT=$((60 * 60))  # 60 minutes
image_list="$(mktemp)"
populate_stackrox_image_list "${image_list}"
info "Will poll for: $(awk '{print $1}' "${image_list}")"

poll_for_system_test_images "${POLL_TIMEOUT}"

# Export the tag for stackrox-install-helm to use
if [[ -n "${SHARED_DIR:-}" ]]; then
  echo "${ACS_VERSION_TAG}" > "${SHARED_DIR}/acs_image_tag"
  info "Exported ACS_IMAGE_TAG=${ACS_VERSION_TAG} to ${SHARED_DIR}/acs_image_tag"
else
  die "ERROR: SHARED_DIR is not set, cannot export tag for installation step"
fi

info "PR images are ready for installation"

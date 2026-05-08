#!/usr/bin/env bash

# Wait for PR images to be available for performance/scale testing.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
# shellcheck source=../lib.sh
source "$ROOT/scripts/ci/lib.sh"

info "Waiting for PR images for performance/scale testing"

MAIN_IMAGE_TAG="$(make tag)"
info "Target image tag: ${MAIN_IMAGE_TAG}"

image_list="$(mktemp)"
populate_stackrox_image_list "${image_list}"
info "Will poll for: $(awk '{print $1}' "${image_list}")"

poll_for_system_test_images 3600

# Export the tag for stackrox-install-helm to use
if [[ -n "${SHARED_DIR:-}" ]]; then
  echo "${MAIN_IMAGE_TAG}" > "${SHARED_DIR}/acs_image_tag"
  info "Exported ACS_IMAGE_TAG=${MAIN_IMAGE_TAG} to ${SHARED_DIR}/acs_image_tag"
else
  die "ERROR: SHARED_DIR is not set, cannot export tag for installation step"
fi

info "PR images are ready for installation"

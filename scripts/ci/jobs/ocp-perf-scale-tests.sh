#!/usr/bin/env bash

# Wait for PR images to be available for performance/scale testing.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
# shellcheck source=../lib.sh
source "$ROOT/scripts/ci/lib.sh"

# === PERF/SCALE VERSION BISECTION OVERRIDE ===
# To test a specific, already-published ACS version instead of this branch's
# images, set this to a published image tag. Any merged master commit has
# images at quay.io/stackrox-io/main:<tag> where <tag> is that commit's
# `make tag` value (format: X.Y.x-N-gSHA); nightly tags (X.Y.x-nightly-YYYYMMDD)
# work too. roxie just pulls this tag, so no rebuild or code revert is needed.
# Leave empty for normal behavior (poll for this branch's freshly built images).
ACS_IMAGE_TAG_OVERRIDE="${ACS_IMAGE_TAG_OVERRIDE:-5.1.x-529-g9501cace08}"

if [[ -n "${ACS_IMAGE_TAG_OVERRIDE}" ]]; then
  MAIN_IMAGE_TAG="${ACS_IMAGE_TAG_OVERRIDE}"
  info "ACS_IMAGE_TAG_OVERRIDE set: deploying ${MAIN_IMAGE_TAG} (skipping PR image build/poll)"
else
  info "Waiting for PR images for performance/scale testing"

  MAIN_IMAGE_TAG="$(make tag)"
  info "Target image tag: ${MAIN_IMAGE_TAG}"

  image_list="$(mktemp)"
  populate_stackrox_image_list "${image_list}"
  info "Will poll for: $(awk '{print $1}' "${image_list}")"

  poll_for_system_test_images 3600
fi

# Export the tag for stackrox-install-roxie to use
if [[ -n "${SHARED_DIR:-}" ]]; then
  echo "${MAIN_IMAGE_TAG}" > "${SHARED_DIR}/acs_image_tag"
  info "Exported ACS_IMAGE_TAG=${MAIN_IMAGE_TAG} to ${SHARED_DIR}/acs_image_tag"
else
  die "ERROR: SHARED_DIR is not set, cannot export tag for installation step"
fi

info "PR images are ready for installation"

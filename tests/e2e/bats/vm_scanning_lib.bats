#!/usr/bin/env bats

# This file contains tests for the helpers in vm-scanning-lib.sh that resolve
# the image used for the guest roxagent Quadlet.
#
# This test suite can be run using:
#
#   bats tests/e2e/bats/vm_scanning_lib.bats

# shellcheck disable=SC1091
load "../../../scripts/test_helpers.bats"

function setup() {
    unset USE_KONFLUX_IMAGES
    source "${BATS_TEST_DIRNAME}/../vm-scanning-lib.sh"
}

# -- _vm_scanning_main_image_name --

@test "_vm_scanning_main_image_name defaults to main when USE_KONFLUX_IMAGES is unset" {
    run _vm_scanning_main_image_name
    assert_success
    assert_output "main"
}

@test "_vm_scanning_main_image_name uses release-main for Konflux images" {
    USE_KONFLUX_IMAGES=true run _vm_scanning_main_image_name
    assert_success
    assert_output "release-main"
}

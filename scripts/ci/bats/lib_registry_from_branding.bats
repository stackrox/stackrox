#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/../lib.sh"
}

@test "registry_from_branding() STACKROX_BRANDING" {
    run registry_from_branding "STACKROX_BRANDING"
    assert_success
    assert_output "quay.io/stackrox-io"
}

@test "registry_from_branding() RHACS_BRANDING" {
    run registry_from_branding "RHACS_BRANDING"
    assert_success
    assert_output "quay.io/rhacs-eng"
}

@test "registry_from_branding() no argument" {
    run registry_from_branding
    assert_failure
}

@test "registry_from_branding() unknown branding" {
    run registry_from_branding "cabbage"
    assert_failure
    assert_output --partial "not a supported brand"
}

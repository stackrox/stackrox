#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/../lib.sh"
}

# is_{release,RC}_version() helper tests

@test "is_release_version() expects an arg" {
    run is_release_version
    assert_failure 1
    assert_output --partial 'missing arg'
}

@test "is_release_version() is a release" {
    run is_release_version "3.67.2"
    assert_success
}

@test "is_release_version() an RC is not a release" {
    run is_release_version "3.67.2-rc.1"
    assert_failure 1
}

@test "is_release_version() a dev build is not a release" {
    run is_release_version "3.68.x-23-g8a2e05d0ec"
    assert_failure 1
}

@test "is_RC_version() expects an arg" {
    run is_RC_version
    assert_failure 1
    assert_output --partial 'missing arg'
}

@test "is_RC_version() is an RC" {
    run is_RC_version "3.67.2-rc.2"
    assert_success
}

@test "is_RC_version() a release is not an RC" {
    run is_RC_version "3.67.2"
    assert_failure 1
}

@test "is_RC_version() a dev build is not an RC" {
    run is_RC_version "3.68.x-23-g8a2e05d0ec"
    assert_failure 1
}

# check_scanner_and_collector() tests

function make() {
    echo "${tags[$2]}"
}

@test "spots unreleased collector tags when an RC" {
    declare -A tags=( [tag]="3.67.2-rc.2" [collector-tag]="3.68.x-23-g8a2e05d0ec" [scanner-tag]="3.45.1")
    run check_scanner_and_collector "false"
    assert_success
    assert_output --partial 'Collector tag does not look like a release tag'
    refute_output --partial 'Scanner tag does not look like a release tag'
}

@test "spots unreleased scanner tags when an RC" {
    declare -A tags=( [tag]="3.67.2-rc.2" [collector-tag]="3.68.1" [scanner-tag]="3.45.1-rc.2")
    run check_scanner_and_collector "false"
    assert_success
    refute_output --partial 'Collector tag does not look like a release tag'
    assert_output --partial 'Scanner tag does not look like a release tag'
}

@test "spots unreleased collector tags when an RC and can fail" {
    declare -A tags=( [tag]="3.67.2-rc.2" [collector-tag]="3.68.x-23-g8a2e05d0ec" [scanner-tag]="3.45.1")
    run check_scanner_and_collector "true"
    assert_failure
    assert_output --partial 'Collector tag does not look like a release tag'
    refute_output --partial 'Scanner tag does not look like a release tag'
}

@test "spots unreleased scanner tags when an RC and can fail" {
    declare -A tags=( [tag]="3.67.2-rc.2" [collector-tag]="3.68.1" [scanner-tag]="3.45.1-rc.2")
    run check_scanner_and_collector "true"
    assert_failure
    refute_output --partial 'Collector tag does not look like a release tag'
    assert_output --partial 'Scanner tag does not look like a release tag'
}

@test "spots unreleased collector tags when a release and fails" {
    declare -A tags=( [tag]="3.67.2" [collector-tag]="3.68.23-rc.8" [scanner-tag]="3.45.1")
    run check_scanner_and_collector "false"
    assert_failure
    assert_output --partial 'Collector tag does not look like a release tag'
    refute_output --partial 'Scanner tag does not look like a release tag'
}

@test "spots unreleased scanner tags when a release and fails" {
    declare -A tags=( [tag]="3.67.2" [collector-tag]="3.68.1" [scanner-tag]="3.45.x-23-g8a2e05d0ec")
    run check_scanner_and_collector "false"
    assert_failure
    refute_output --partial 'Collector tag does not look like a release tag'
    assert_output --partial 'Scanner tag does not look like a release tag'
}

@test "spots both unreleased tags when a release and fails" {
    declare -A tags=( [tag]="3.67.2" [collector-tag]="3.68.23-rc.8" [scanner-tag]="3.45.x-23-g8a2e05d0ec")
    run check_scanner_and_collector "false"
    assert_failure
    assert_output --partial 'Collector tag does not look like a release tag'
    assert_output --partial 'Scanner tag does not look like a release tag'
}

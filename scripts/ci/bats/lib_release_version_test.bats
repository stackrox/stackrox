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

@test "get_release_stream() gives major.minor I" {
    run get_release_stream "3.67.2-rc.2"
    assert_success
    assert_output "3.67"
}

@test "get_release_stream() gives major.minor II" {
    run get_release_stream "3.68.x-23-g8a2e05d0ec"
    assert_success
    assert_output "3.68"
}

@test "is_release_test_stream() is" {
    run is_release_test_stream "0.0.1-rc2"
    assert_success
    assert_output ""
}

@test "is_release_test_stream() is not" {
    run is_release_test_stream "1.0.1"
    assert_failure
    assert_output ""
}

# check_collector_version(), check_scanner_version() tests

function make() {
    echo "${tags[$3]}"
}

@test "spots collector tag is a master commit" {
    declare -A tags=( [collector-tag]="3.68.x-23-g8a2e05d0ec")
    run check_collector_version
    assert_failure
    assert_output --partial 'Collector tag does not look like a release tag'
}

@test "spots collector tag is a release candidate" {
    declare -A tags=( [collector-tag]="3.68.1-rc.1")
    run check_collector_version
    assert_failure
    assert_output --partial 'Collector tag does not look like a release tag'
}

@test "spots collector tag is a release" {
    declare -A tags=( [collector-tag]="3.68.1")
    run check_collector_version
    assert_success
}

@test "spots scanner tag is a master commit" {
    declare -A tags=( [scanner-tag]="3.45.x-12-g8a2e05d0ec")
    run check_scanner_version
    assert_failure
    assert_output --partial 'Scanner tag does not look like a release tag'
}

@test "spots scanner tag is a release candidate" {
    declare -A tags=( [scanner-tag]="3.45.1-rc.1")
    run check_scanner_version
    assert_failure
    assert_output --partial 'Scanner tag does not look like a release tag'
}

@test "spots scanner tag is a release" {
    declare -A tags=( [scanner-tag]="3.45.1")
    run check_scanner_version
    assert_success
}

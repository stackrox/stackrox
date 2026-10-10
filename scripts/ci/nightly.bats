#!/usr/bin/env bats

load "../test_helpers.bats"

setup() {
    # shellcheck source=./nightly.sh
    source "${BATS_TEST_DIRNAME}/nightly.sh"
    unset BUILD_TAG GITHUB_REF
}

@test "nightly detection accepts build tags and GitHub refs" {
    BUILD_TAG=4.11.x-nightly-20261001 run is_nightly_run
    assert_success
    GITHUB_REF=refs/tags/4.11.x-nightly-20261001 run is_nightly_run
    assert_success
}

@test "PR, release and local runs are not nightlies" {
    run is_nightly_run
    assert_failure
    GITHUB_REF=refs/pull/22596/merge run is_nightly_run
    assert_failure
    BUILD_TAG=4.11.0 GITHUB_REF=refs/tags/4.11.0 run is_nightly_run
    assert_failure
}

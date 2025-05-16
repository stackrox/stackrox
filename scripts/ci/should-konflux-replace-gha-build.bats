#!/usr/bin/env bats

load "../test_helpers.bats"

function setup() {
    unset SOURCE_BRANCH
    unset TARGET_BRANCH
    unset GITHUB_BASE_REF
    unset GITHUB_REF
}

function run_cmd() {
    run --separate-stderr "${BATS_TEST_DIRNAME}/should-konflux-replace-gha-build.sh"
}

function check_both_go() {
    run_cmd
    assert_success
    assert_output "BUILD_AND_PUSH_BOTH"
    assert_stderr_contains "does not look like"
    assert_stderr_contains "release"
    assert_stderr_contains "branch or tag"
}

function check_gha_suppressed() {
    run_cmd
    assert_success
    assert_output "BUILD_AND_PUSH_ONLY_KONFLUX"
    assert_stderr_contains "looks like"
    assert_stderr_contains "release"
    assert_stderr_contains "branch or tag"
}

# BATS libraries in our builder image don't have assert_stderr.
function assert_stderr_contains() {
    assert grep -F "$1" <<< "${stderr_lines[@]}"
}

@test "should fail when no values are set" {
    run_cmd
    assert_failure 2
}

# When executing in Konflux

@test "should tell only Konflux when rc tag pushed" {
    export SOURCE_BRANCH=refs/tags/4.10.56-rc.172
    export TARGET_BRANCH=refs/tags/4.10.56-rc.172
    check_gha_suppressed
}

@test "should tell Both when a different tag pushed" {
    export SOURCE_BRANCH=refs/tags/4.10.56-nightly.20250515
    export TARGET_BRANCH=refs/tags/4.10.56-nightly.20250515
    check_both_go
}

@test "should tell only Konflux when release-like branch pushed" {
    export SOURCE_BRANCH=release-4.8
    export TARGET_BRANCH=release-4.8
    check_gha_suppressed
}

@test "should tell Both when non-release branch pushed" {
    export SOURCE_BRANCH=author/ROX-27716-useful-feature
    export TARGET_BRANCH=author/ROX-27716-useful-feature
    check_both_go
}

# When executing in GHA

@test "should tell only Konflux when github_ref is release tag" {
    export GITHUB_REF=refs/tags/24.58.60
    check_gha_suppressed
}

@test "should tell Both when github_ref is a different tag" {
    export GITHUB_REF=refs/tags/0.0.0-author-testing
    check_both_go
}

@test "should tell only Konflux when github_ref is release-like" {
    export GITHUB_REF=refs/heads/release-x.y
    check_gha_suppressed
}

@test "should tell Both when github_ref is other" {
    export GITHUB_REF=refs/heads/many-funky/parts/with-useful/slashes
    check_both_go
}

@test "should tell only Konflux when PR and github_base_ref is release-like" {
    export GITHUB_REF="refs/pull/15309/merge"
    export GITHUB_BASE_REF="release-x.y"
    check_gha_suppressed
}

@test "should tell Both when PR and github_base_ref is other" {
    export GITHUB_REF="refs/pull/15309/merge"
    export GITHUB_BASE_REF="master"
    check_both_go
}

@test "should fail when GITHUB_BASE_REF should be set but it's not" {
    export GITHUB_REF="refs/pull/1005006/merge"
    run_cmd
    assert_failure
}

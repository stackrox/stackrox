#!/usr/bin/env bats

load "../test_helpers.bats"

CMD="${BATS_TEST_DIRNAME}/build-in-konflux-instead-of-gha.sh"

function setup() {
    unset TARGET_BRANCH
    unset GITHUB_REF
}

function run_cmd() {
    run --separate-stderr "${CMD}"
}

function check_both_build() {
    run_cmd
    assert_success
    assert_output "build and push both"
    assert_stderr_contains "does not look like"
    assert_stderr_contains "release"
    assert_stderr_contains "branch or tag"
}

function check_gha_suppressed() {
    run_cmd
    assert_success
    assert_output "build and push only Konflux"
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

@test "should build only in Konflux when TARGET_BRANCH is release-like" {
    export TARGET_BRANCH=release-4.8
    check_gha_suppressed
}

@test "should build only in Konflux when github_ref is release-like" {
    export GITHUB_REF=refs/heads/release-x.y
    check_gha_suppressed
}

@test "should build both GHA and Konflux when TARGET_BRANCH is other" {
    export TARGET_BRANCH=author/ROX-27716-take-konflux-on-release
    check_both_build
}

@test "should build both GHA and Konflux when github_ref is other" {
    export GITHUB_REF=refs/heads/many-funky/components/with-useful/slashes
    check_both_build
}

@test "should build only in Konflux when TARGET_BRANCH is rc tag" {
    export TARGET_BRANCH=refs/tags/4.10.56-rc.172
    check_gha_suppressed
}

@test "should build only in Konflux when github_ref is release tag" {
    export GITHUB_REF=refs/tags/24.58.60
    check_gha_suppressed
}

@test "should build both GHA and Konflux when TARGET_BRANCH is a different tag" {
    export TARGET_BRANCH=refs/tags/4.10.56-nightly.20250515
    check_both_build
}

@test "should build both GHA and Konflux when github_ref is a different tag" {
    export GITHUB_REF=refs/tags/author-testing
    check_both_build
}

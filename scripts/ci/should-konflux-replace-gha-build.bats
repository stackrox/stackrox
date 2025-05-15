#!/usr/bin/env bats

load "../test_helpers.bats"

function setup() {
    unset SOURCE_BRANCH
    unset TARGET_BRANCH
    unset GITHUB_BASE_REF
    unset GITHUB_HEAD_REF
    unset GITHUB_REF
    bats_require_minimum_version 1.5.0
}

function run_cmd() {
    # We copy the script to a temporary directory and run it from there so that it does not find
    # should-konflux-replace-gha-build.hold file if that's present in the repo.

    cp -a "${BATS_TEST_DIRNAME}/should-konflux-replace-gha-build.sh" "${BATS_TEST_TMPDIR}/our-script.sh"
    run --separate-stderr "${BATS_TEST_TMPDIR}/our-script.sh"
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

function check_gha_suppressed_for_pr() {
    run_cmd
    assert_success
    assert_output "BUILD_AND_PUSH_ONLY_KONFLUX"
    assert_stderr_contains "magic"
}

# BATS libraries in our builder image don't have assert_stderr.
function assert_stderr_contains() {
    assert grep -F "$1" <<< "${stderr_lines[@]}"
}

@test "should fail when required values are not set" {
    run_cmd
    assert_failure 2
}

# When executing in Konflux

@test "Konflux: should tell only Konflux when rc tag pushed" {
    export SOURCE_BRANCH=refs/tags/4.10.56-rc.172
    export TARGET_BRANCH=refs/tags/4.10.56-rc.172
    check_gha_suppressed
}

@test "Konflux: should tell Both when a different tag pushed" {
    export SOURCE_BRANCH=refs/tags/4.10.56-nightly.20250515
    export TARGET_BRANCH=refs/tags/4.10.56-nightly.20250515
    check_both_go
}

@test "Konflux: should tell only Konflux when release-like branch pushed" {
    export SOURCE_BRANCH=release-4.8
    export TARGET_BRANCH=release-4.8
    check_gha_suppressed
}

@test "Konflux: should tell Both when non-release branch pushed" {
    export SOURCE_BRANCH=author/ROX-27716-useful-feature
    export TARGET_BRANCH=author/ROX-27716-useful-feature
    check_both_go
}

@test "Konflux: should tell only Konflux when PR branch name includes magic" {
    export SOURCE_BRANCH=author/konflux-release-like
    export TARGET_BRANCH=master
    check_gha_suppressed_for_pr
}

@test "Konflux: should tell Both when PR branch name is not magic" {
    export SOURCE_BRANCH=author/my-useful-feature
    export TARGET_BRANCH=master
    check_both_go
}

@test "Konflux: should tell only Konflux when PR targets release branch" {
    export SOURCE_BRANCH=author/my-useful-feature
    export TARGET_BRANCH=release-4.8
    check_gha_suppressed
}

# When executing in GHA

@test "GHA: should tell only Konflux when release tag pushed" {
    export GITHUB_REF=refs/tags/24.58.60
    check_gha_suppressed
}

@test "GHA: should tell Both when different tag pushed" {
    export GITHUB_REF=refs/tags/0.0.0-author-testing
    check_both_go
}

@test "GHA: should tell only Konflux when release-like branch pushed" {
    export GITHUB_REF=refs/heads/release-x.y
    check_gha_suppressed
}

@test "GHA: should tell Both when non-release branch pushed" {
    export GITHUB_REF=refs/heads/many-funky/parts/with-useful/slashes
    check_both_go
}

@test "GHA: should fail when PR but variables are not set" {
    export GITHUB_REF=refs/pull/1005006/merge
    run_cmd
    assert_failure 3
}

@test "GHA: should tell only Konflux when PR targets release-like branch" {
    export GITHUB_REF=refs/pull/15309/merge
    export GITHUB_BASE_REF=release-x.y
    export GITHUB_HEAD_REF=author/my-useful-feature
    check_gha_suppressed
}

@test "GHA: should tell Both when PR targets non-release branch" {
    export GITHUB_REF=refs/pull/15309/merge
    export GITHUB_BASE_REF=master
    export GITHUB_HEAD_REF=author/my-useful-feature
    check_both_go
}

@test "GHA: should tell only Konflux when PR branch name includes magic" {
    export GITHUB_REF=refs/pull/15309/merge
    export GITHUB_BASE_REF=master
    export GITHUB_HEAD_REF=author/konflux-release-like
    check_gha_suppressed_for_pr
}

# Holdfile logic

@test "should respect holdfile when release push" {
    export SOURCE_BRANCH=release-4.8
    export TARGET_BRANCH=release-4.8
    create_holdfile
    run_cmd
    assert_success
    assert_output "BUILD_AND_PUSH_BOTH"
    assert_stderr_contains "holdfile"
}

@test "should ignore holdfile when PR with magic branch" {
    export SOURCE_BRANCH=author/konflux-release-like
    export TARGET_BRANCH=master
    create_holdfile
    check_gha_suppressed_for_pr
}

function create_holdfile() {
    touch "${BATS_TEST_TMPDIR}/should-konflux-replace-gha-build.hold"
}

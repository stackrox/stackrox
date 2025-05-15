#!/usr/bin/env bats

load "../test_helpers.bats"

CMD="${BATS_TEST_DIRNAME}/build-in-konflux-instead-of-gha.sh"

function setup() {
    unset SOURCE_BRANCH
    unset GITHUB_REF
}

function run_cmd() {
    run "${CMD}"
}

function check_independent() {
    run_cmd
    assert_failure 6
    assert_output --partial 'does not look like'
    assert_output --partial 'release'
    assert_output --partial 'branch or tag'
}

function check_gha_suppressed() {
    run_cmd
    assert_success
    assert_output --partial 'looks like'
    assert_output --partial 'release'
    assert_output --partial 'branch or tag'
}

@test "should fail when no values are set" {
    run_cmd
    assert_failure 2
}

@test "should build only in Konflux when source_branch is release-like" {
    export SOURCE_BRANCH=release-4.8
    check_gha_suppressed
}

@test "should build only in Konflux when github_ref is release-like" {
    export GITHUB_REF=refs/heads/release-x.y
    check_gha_suppressed
}

@test "should build both GHA and Konflux when source_branch is other" {
    export SOURCE_BRANCH=author/ROX-27716-take-konflux-on-release
    check_independent
}

@test "should build both GHA and Konflux when github_ref is other" {
    export GITHUB_REF=refs/heads/many-funky/components/with-useful/slashes
    check_independent
}

@test "should build only in Konflux when source_branch is rc tag" {
    export SOURCE_BRANCH=refs/tags/4.10.56-rc.172
    check_gha_suppressed
}

@test "should build only in Konflux when github_ref is release tag" {
    export GITHUB_REF=refs/tags/24.58.60
    check_gha_suppressed
}

@test "should build both GHA and Konflux when source_branch is a different tag" {
    export SOURCE_BRANCH=refs/tags/4.10.56-nightly.20250515
    check_independent
}

@test "should build both GHA and Konflux when github_ref is a different tag" {
    export GITHUB_REF=refs/tags/author-testing
    check_independent
}

@test "should not produce any stdout" {
    # The command should not print to stdout so that it can be used in places where stdout is used for passing data.

    export GITHUB_REF=refs/heads/ordinary-branch
    run --separate-stderr "${CMD}"
    assert_failure 6
    assert_output ''
    assert [ "${#stderr_lines[@]}" -gt 0 ]

    export GITHUB_REF=refs/heads/release-4.8
    run --separate-stderr "${CMD}"
    assert_success
    assert_output ''
    assert [ "${#stderr_lines[@]}" -gt 0 ]
}

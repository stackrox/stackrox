#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    unset OPENSHIFT_CI
    unset GITHUB_REF
    export GITHUB_ENV="$(mktemp)"
    source "${BATS_TEST_DIRNAME}/../lib.sh"
}

function cleanup() {
    rm -f "$GITHUB_ENV"
}

@test "without any env" {
    run handle_gha_tagged_build
    assert_success
    assert_output --partial 'No GITHUB_REF in env'
    assert_equal "$(cat $GITHUB_ENV)" ""
}

@test "with a ref that does not indicate a tagged build" {
    export GITHUB_REF="refs/heads/nightlies"
    run handle_gha_tagged_build
    assert_success
    assert_output --partial 'This is not a tagged build'
    assert_equal "$(cat $GITHUB_ENV)" ""
}

@test "with a highly unusual ref that might incorrectly indicate a tagged build" {
    export GITHUB_REF="refs/remotes/origin/refs/tags/something"
    run handle_gha_tagged_build
    assert_success
    assert_output --partial 'This is not a tagged build'
    assert_equal "$(cat $GITHUB_ENV)" ""
}

@test "with a ref that indicates a tagged build" {
    export GITHUB_REF="refs/tags/3.73.x-nightly-20221221"
    run handle_gha_tagged_build
    assert_success
    assert_output --partial 'This is a tagged build: 3.73.x-nightly-20221221'
    assert_equal "$(cat $GITHUB_ENV)" "BUILD_TAG=3.73.x-nightly-20221221"
}

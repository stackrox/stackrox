#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/../lib.sh"
}

@test "missing tag argument" {
    run check_docs
    assert_failure 1
    assert_output --partial 'missing arg'
}

@test "exits early if not a release or RC" {
    run check_docs "3.69.x-140-g53e40a27ff"
    assert_success
    assert_output --partial 'Skipping'
    assert_output --partial 'not a release or RC'
}

function git_matches() {
    if [[ "$1" == "config" ]]; then
        echo "rhacs-docs-3.69.0"
    fi
    return 0
}

@test "succeeds when versions match (RC)" {
    function git() {
        git_matches "$@"
    }
    run check_docs "3.69.0-rc.10"
    assert_success
    assert_output --partial 'The docs version is as expected'
}

@test "succeeds when versions match (release)" {
    function git() {
        git_matches "$@"
    }
    run check_docs "3.69.0"
    assert_success
    assert_output --partial 'The docs version is as expected'
}

@test "catches version mismatch (RC)" {
    function git() {
        git_matches "$@"
    }
    run check_docs "4.69.0-rc.10"
    assert_failure
    assert_output --partial 'Expected docs/content'
}

@test "catches version mismatch (release)" {
    function git() {
        git_matches "$@"
    }
    run check_docs "4.69.0"
    assert_failure
    assert_output --partial 'Expected docs/content'
}

function git_matches_but_diffs() {
    if [[ "$1" == "config" ]]; then
        echo "rhacs-docs-3.69.0"
    fi
    if [[ "$1" == "diff" ]]; then
        return 2
    fi
}

@test "catches an out of sync submodule" {
    function git() {
        git_matches_but_diffs "$@"
    }
    run check_docs "3.69.0-rc.10"
    assert_failure 1
    assert_output --partial 'The docs/content submodule is out of date'
}

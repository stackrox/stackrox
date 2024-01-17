#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    unset OPENSHIFT_CI
    unset REPO_OWNER
    unset REPO_NAME
    unset CLONEREFS_OPTIONS
    source "${BATS_TEST_DIRNAME}/../lib.sh"
}

@test "without any env" {
    if [[ -n "${GITHUB_ACTION:-}" ]]; then
        skip "not working on GHA"
    fi
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'unsupported'
}

@test "OPENSHIFT_CI but nothing else" {
    if [[ -n "${GITHUB_ACTION:-}" ]]; then
        skip "not working on GHA"
    fi
    export OPENSHIFT_CI=true
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'Expect REPO_OWNER/NAME or CLONEREFS_OPTIONS'
}

@test "REPO_OWNER without REPO_NAME" {
    if [[ -n "${GITHUB_ACTION:-}" ]]; then
        skip "not working on GHA"
    fi
    export OPENSHIFT_CI=true
    export REPO_OWNER="acme"
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'expect: REPO_NAME'
}

@test "with REPO_OWNER & NAME" {
    if [[ -n "${GITHUB_ACTION:-}" ]]; then
        skip "not working on GHA"
    fi
    export OPENSHIFT_CI=true
    export REPO_OWNER="acme"
    export REPO_NAME="products"
    run get_repo_full_name
    assert_success
    assert_output 'acme/products'
}

@test "with invalid CLONEREFS_OPTIONS I" {
    if [[ -n "${GITHUB_ACTION:-}" ]]; then
        skip "not working on GHA"
    fi
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ }'
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'expect: org and repo'
}

@test "with invalid CLONEREFS_OPTIONS II" {
    if [[ -n "${GITHUB_ACTION:-}" ]]; then
        skip "not working on GHA"
    fi
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ "refs": [] }'
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'expect: org and repo'
}

@test "with invalid CLONEREFS_OPTIONS III" {
    if [[ -n "${GITHUB_ACTION:-}" ]]; then
        skip "not working on GHA"
    fi
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ "refs": [{ "org": "acme" }] }'
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'expect: org and repo'
}

@test "with invalid CLONEREFS_OPTIONS IV" {
    if [[ -n "${GITHUB_ACTION:-}" ]]; then
        skip "not working on GHA"
    fi
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ "not yamls" }'
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'invalid CLONEREFS_OPTIONS yaml'
}

@test "with invalid CLONEREFS_OPTIONS V" {
    if [[ -n "${GITHUB_ACTION:-}" ]]; then
        skip "not working on GHA"
    fi
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ "refs": "" }'
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'invalid CLONEREFS_OPTIONS yaml'
}

@test "with valid CLONEREFS_OPTIONS" {
    if [[ -n "${GITHUB_ACTION:-}" ]]; then
        skip "not working on GHA"
    fi
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ "refs": [{ "org": "acme", "repo": "products" }] }'
    run get_repo_full_name
    assert_success
    assert_output 'acme/products'
}


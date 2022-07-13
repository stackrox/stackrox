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
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'unsupported'
}

@test "OPENSHIFT_CI but nothing else" {
    export OPENSHIFT_CI=true
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'Expect REPO_OWNER/NAME or CLONEREFS_OPTIONS'
}

@test "REPO_OWNER without REPO_NAME" {
    export OPENSHIFT_CI=true
    export REPO_OWNER="acme"
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'expect: REPO_NAME'
}

@test "with REPO_OWNER & NAME" {
    export OPENSHIFT_CI=true
    export REPO_OWNER="acme"
    export REPO_NAME="products"
    run get_repo_full_name
    assert_success
    assert_output 'acme/products'
}

@test "with invalid CLONEREFS_OPTIONS I" {
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ }'
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'expect: org and repo'
}

@test "with invalid CLONEREFS_OPTIONS II" {
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ "refs": [] }'
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'expect: org and repo'
}

@test "with invalid CLONEREFS_OPTIONS III" {
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ "refs": [{ "org": "acme" }] }'
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'expect: org and repo'
}

@test "with invalid CLONEREFS_OPTIONS IV" {
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ "not yamls" }'
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'invalid CLONEREFS_OPTIONS yaml'
}

@test "with invalid CLONEREFS_OPTIONS V" {
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ "refs": "" }'
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'invalid CLONEREFS_OPTIONS yaml'
}

@test "with valid CLONEREFS_OPTIONS" {
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ "refs": [{ "org": "acme", "repo": "products" }] }'
    run get_repo_full_name
    assert_success
    assert_output 'acme/products'
}


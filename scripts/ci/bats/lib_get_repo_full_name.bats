#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
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
    assert_output --partial 'Expect REPO_OWNER/NAME or JOB_SPEC'
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

@test "with invalid JOB_SPEC I" {
    export OPENSHIFT_CI=true
    export JOB_SPEC='{ }'
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'expect: org and repo'
}

@test "with invalid JOB_SPEC II" {
    export OPENSHIFT_CI=true
    export JOB_SPEC='{ "extra_refs": [] }'
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'expect: org and repo'
}

@test "with invalid JOB_SPEC III" {
    export OPENSHIFT_CI=true
    export JOB_SPEC='{ "extra_refs": [{ "org": "acme" }] }'
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'expect: org and repo'
}

@test "with invalid JOB_SPEC IV" {
    export OPENSHIFT_CI=true
    export JOB_SPEC='{ "not yamls" }'
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'invalid JOB_SPEC yaml'
}

@test "with invalid JOB_SPEC V" {
    export OPENSHIFT_CI=true
    export JOB_SPEC='{ "extra_refs": "" }'
    run get_repo_full_name
    assert_failure 1
    assert_output --partial 'invalid JOB_SPEC yaml'
}

@test "with valid JOB_SPEC" {
    export OPENSHIFT_CI=true
    export JOB_SPEC='{ "extra_refs": [{ "org": "acme", "repo": "products" }] }'
    run get_repo_full_name
    assert_success
    assert_output 'acme/products'
}


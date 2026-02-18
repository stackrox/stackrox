#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    unset OPENSHIFT_CI
    unset PULL_BASE_REF
    unset PULL_HEAD_REF
    unset CLONEREFS_OPTIONS
    unset GITHUB_ACTION
    unset GITHUB_HEAD_REF
    unset GITHUB_REF_NAME
    source "${BATS_TEST_DIRNAME}/../lib.sh"
}

@test "without any env" {
    run get_branch_name
    assert_failure 1
    assert_output --partial 'unsupported'
}

@test "prow but nothing else" {
    export OPENSHIFT_CI=true
    run get_branch_name
    assert_failure 1
    assert_output --partial 'ERROR: Expected'
}

@test "prow with PULL_HEAD_REF" {
    export OPENSHIFT_CI=true
    export PULL_HEAD_REF="mrsmith/fix-everything"
    run get_branch_name
    assert_success
    assert_output 'mrsmith/fix-everything'
}

@test "prow with PULL_BASE_REF" {
    export OPENSHIFT_CI=true
    export PULL_BASE_REF="main"
    run get_branch_name
    assert_success
    assert_output 'main'
}

@test "prow with invalid CLONEREFS_OPTIONS I" {
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{}'
    run get_branch_name
    assert_failure 1
    assert_output --partial 'expect: base_ref'
}

@test "prow with invalid CLONEREFS_OPTIONS II" {
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ "refs": [] }'
    run get_branch_name
    assert_failure 1
    assert_output --partial 'expect: base_ref'
}

@test "prow with invalid CLONEREFS_OPTIONS III" {
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ "not yamls" }'
    run get_branch_name
    assert_failure 1
    assert_output --partial 'invalid CLONEREFS_OPTIONS yaml'
}

@test "prow with invalid CLONEREFS_OPTIONS IV" {
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ "refs": "" }'
    run get_branch_name
    assert_failure 1
    assert_output --partial 'invalid CLONEREFS_OPTIONS yaml'
}

@test "prow with valid CLONEREFS_OPTIONS" {
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{ "refs": [{ "base_ref": "main" }] }'
    run get_branch_name
    assert_success
    assert_output 'main'
}

@test "GHA without any env" {
    export GITHUB_ACTION=true
    run get_branch_name
    assert_failure 1
    assert_output --partial 'ERROR: Expected'
}

@test "GHA with both refs" {
    export GITHUB_ACTION=true
    export GITHUB_HEAD_REF="mrsmith/fix-everything"
    export GITHUB_REF_NAME="master"
    run get_branch_name
    assert_success
    assert_output 'mrsmith/fix-everything'
}

@test "GHA with only GITHUB_REF_NAME" {
    export GITHUB_ACTION=true
    export GITHUB_REF_NAME="master"
    run get_branch_name
    assert_success
    assert_output 'master'
}

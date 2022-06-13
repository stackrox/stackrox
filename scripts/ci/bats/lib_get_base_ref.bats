#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/../lib.sh"
}

@test "without any env" {
    run get_base_ref
    assert_failure 1
    assert_output --partial 'unsupported'
}

@test "OPENSHIFT_CI but nothing else" {
    export OPENSHIFT_CI=true
    run get_base_ref
    assert_failure 1
    assert_output --partial 'Expect PULL_BASE_REF or JOB_SPEC'
}

@test "with PULL_BASE_REF" {
    export OPENSHIFT_CI=true
    export PULL_BASE_REF="main"
    run get_base_ref
    assert_success
    assert_output 'main'
}

@test "with invalid JOB_SPEC I" {
    export OPENSHIFT_CI=true
    export JOB_SPEC='{}'
    run get_base_ref
    assert_failure 1
    assert_output --partial 'expect: base_ref'
}

@test "with invalid JOB_SPEC II" {
    export OPENSHIFT_CI=true
    export JOB_SPEC='{ "extra_refs": [] }'
    run get_base_ref
    assert_failure 1
    assert_output --partial 'expect: base_ref'
}

@test "with invalid JOB_SPEC III" {
    export OPENSHIFT_CI=true
    export JOB_SPEC='{ "not yamls" }'
    run get_base_ref
    assert_failure 1
    assert_output --partial 'invalid JOB_SPEC yaml'
}

@test "with invalid JOB_SPEC IV" {
    export OPENSHIFT_CI=true
    export JOB_SPEC='{ "extra_refs": "" }'
    run get_base_ref
    assert_failure 1
    assert_output --partial 'invalid JOB_SPEC yaml'
}

@test "with valid JOB_SPEC" {
    export OPENSHIFT_CI=true
    export JOB_SPEC='{ "extra_refs": [{ "base_ref": "main" }] }'
    run get_base_ref
    assert_success
    assert_output 'main'
}


#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    export CI=true
    export OPENSHIFT_CI=true
    source "${BATS_TEST_DIRNAME}/../lib.sh"
}

@test "a pr may not have a pragma" {
    _PR_DETAILS=$(cat "${BATS_TEST_DIRNAME}/fixtures/a_pr.json")
    run pr_has_pragma "repeat"
    assert_failure
    assert_output ''
}

@test "a pr may have a pragma" {
    _PR_DETAILS=$(cat "${BATS_TEST_DIRNAME}/fixtures/a_pr_with_a_pragma.json")
    run pr_has_pragma "gke_release_channel"
    assert_success
    assert_output ''
}

@test "a pragma has a value" {
    _PR_DETAILS=$(cat "${BATS_TEST_DIRNAME}/fixtures/a_pr_with_a_pragma.json")
    run pr_get_pragma "gke_release_channel"
    assert_success
    assert_output 'rapid'
}

@test "trims and supports internal space" {
    _PR_DETAILS=$(cat "${BATS_TEST_DIRNAME}/fixtures/a_pr_with_a_pragma.json")
    run pr_get_pragma "not_fussy"
    assert_success
    assert_output 'a not fussy value'
}

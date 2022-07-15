#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    export CI=true
    export OPENSHIFT_CI=true
    source "${BATS_TEST_DIRNAME}/../lib.sh"
}

@test "a pr may have a label" {
    local pr_details=$(cat "${BATS_TEST_DIRNAME}/fixtures/a_pr.json")
    run pr_has_label "has-this-label" "$pr_details"
    assert_success
    assert_output ''
}

@test "a pr may not have a label" {
    local pr_details=$(cat "${BATS_TEST_DIRNAME}/fixtures/a_pr.json")
    run pr_has_label "does-not-have-this-label" "$pr_details"
    assert_failure
    assert_output ''
}

@test "a pr body may have a label" {
    local pr_details=$(cat "${BATS_TEST_DIRNAME}/fixtures/a_pr.json")
    run pr_has_label_in_body "has-this-label" "$pr_details"
    assert_success
    assert_output ''
}

@test "a pr body may not have a label" {
    local pr_details=$(cat "${BATS_TEST_DIRNAME}/fixtures/a_pr.json")
    run pr_has_label_in_body "does-not-have-this-label" "$pr_details"
    assert_failure
    assert_output ''
}

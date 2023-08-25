#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    export CI=true
    unset OPENSHIFT_CI
    unset PULL_NUMBER
    unset CLONEREFS_OPTIONS
    source "${BATS_TEST_DIRNAME}/../lib.sh"
}

@test "is_in_PR_context() is not by default" {
    run is_in_PR_context
    assert_failure 1
    assert_output ''
}

@test "is_in_PR_context() is not for OpenShift CI" {
    export OPENSHIFT_CI=true
    run is_in_PR_context
    assert_failure 1
    assert_output ''
}

@test "is_in_PR_context() is for OpenShift CI pulls" {
    export OPENSHIFT_CI=true
    export PULL_NUMBER=99
    run is_in_PR_context
    assert_success
    assert_output ''
}

@test "is_in_PR_context() is for OpenShift CI test-bin" {
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{"src_root":"/go","log":"/dev/null","git_user_name":"ci-robot","git_user_email":"ci-robot@openshift.io","refs":[{"org":"stackrox","repo":"stackrox","repo_link":"https://github.com/stackrox/stackrox","base_ref":"master","base_sha":"9827e744730820045fa14935f3cd1858c9a1afad","base_link":"https://github.com/stackrox/stackrox/commit/9827e744730820045fa14935f3cd1858c9a1afad","pulls":[{"number":2224,"author":"gavin-stackrox","sha":"3bfd0d09eb2bf5e8be74e87724512ca714a1f516","link":"https://github.com/stackrox/stackrox/pull/2224","commit_link":"https://github.com/stackrox/stackrox/pull/2224/commits/3bfd0d09eb2bf5e8be74e87724512ca714a1f516","author_link":"https://github.com/gavin-stackrox"}]}],"fail":true}'
    run is_in_PR_context
    assert_success
    assert_output ''
}

@test "is_in_PR_context() is not for OpenShift CI unexpected env" {
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='{"bad":"ness"}'
    run is_in_PR_context
    assert_failure 1
    assert_output ''
}

@test "is_in_PR_context() is not for OpenShift CI unexpected env II" {
    export OPENSHIFT_CI=true
    export CLONEREFS_OPTIONS='real badness'
    run is_in_PR_context
    assert_failure 1
    assert_output ''
}

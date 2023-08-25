#!/usr/bin/env bats

CMD="${BATS_TEST_DIRNAME}/check-restart-logs.sh"
TEST_FIXTURES="${BATS_TEST_DIRNAME}/test_fixtures"

@test "needs 2 args" {
    run "$CMD"
    [ "$status" -eq 1 ]
}

@test "needs 2 args II" {
    run "$CMD" "openshift-crio-api-e2e-tests"
    [ "$status" -eq 1 ]
}

@test "the log should exist" {
    run "$CMD" "openshift-crio-api-e2e-tests" /no-existo
    [ "$status" -eq 1 ]
    [ "$output" = "Error: the log file '/no-existo' does not exist" ]
}

@test "a log with no exception is not OK" {
    run "$CMD" "openshift-crio-api-e2e-tests" "${TEST_FIXTURES}/no-exception-collector-previous.log"
    [ "$status" -eq 2 ]
    [ "${lines[0]}" = "Checking for a restart exception in: ${TEST_FIXTURES}/no-exception-collector-previous.log" ]
    [ "${lines[1]}" = "This restart does not match any ignore patterns" ]
}

@test "a log with an exception is OK" {
    run "$CMD" "openshift-crio-api-e2e-tests" "${TEST_FIXTURES}/exception-collector-previous.log"
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "Checking for a restart exception in: ${TEST_FIXTURES}/exception-collector-previous.log" ]
    [ "${lines[1]}" = "Ignoring this restart due to: collector initialization restart with download failure" ]
}

@test "it can depend on process" {
    run "$CMD" "openshift-crio-api-e2e-tests" "${TEST_FIXTURES}/other-process-previous.log"
    [ "$status" -eq 2 ]
}

@test "it can depend on CI job" {
    run "$CMD" "another-job" "${TEST_FIXTURES}/rox-5861-exception-compliance-previous.log"
    [ "$status" -eq 2 ]
}

@test "it handles the exception for ROX-5861" {
    run "$CMD" "openshift-api-e2e-tests" "${TEST_FIXTURES}/rox-5861-exception-compliance-previous.log"
    [ "$status" -eq 0 ]
}

@test "it handles collector restarts under openshift due to slow sensor start" {
    run "$CMD" "openshift-api-e2e-tests" "${TEST_FIXTURES}/slow-sensor-collector-previous.log"
    [ "$status" -eq 0 ]
}

@test "it only allows this ^^ exception for openshift" {
    run "$CMD" "banana-e2e-tests" "${TEST_FIXTURES}/slow-sensor-collector-previous.log"
    [ "$status" -eq 2 ]
}

@test "it handles exceptions in > 1 logs" {
    run "$CMD" "openshift-api-e2e-tests" "${TEST_FIXTURES}/exception-collector-previous.log" "${TEST_FIXTURES}/rox-5861-exception-compliance-previous.log"
    [ "$status" -eq 0 ]
}

@test "it spots a log with no exceptions with other logs that have an exception (by content)" {
    run "$CMD" "openshift-api-e2e-tests" "${TEST_FIXTURES}/no-exception-collector-previous.log" "${TEST_FIXTURES}/exception-collector-previous.log" "${TEST_FIXTURES}/rox-5861-exception-compliance-previous.log"
    [ "$status" -eq 2 ]
}

@test "it spots a log with no exceptions with other logs that have an exception (by process)" {
    run "$CMD" "openshift-api-e2e-tests" "${TEST_FIXTURES}/other-process-previous.log" "${TEST_FIXTURES}/exception-collector-previous.log" "${TEST_FIXTURES}/rox-5861-exception-compliance-previous.log"
    [ "$status" -eq 2 ]
}

@test "ordering is not a problem" {
    run "$CMD" "openshift-api-e2e-tests" "${TEST_FIXTURES}/exception-collector-previous.log" "${TEST_FIXTURES}/rox-5861-exception-compliance-previous.log" "${TEST_FIXTURES}/no-exception-collector-previous.log"
    [ "$status" -eq 2 ]
}

@test "checks them all" {
    run "$CMD" "openshift-api-e2e-tests" "${TEST_FIXTURES}/exception-collector-previous.log" "${TEST_FIXTURES}/no-exception-collector-previous.log" "${TEST_FIXTURES}/rox-5861-exception-compliance-previous.log"
    [ "$status" -eq 2 ]
    [ "${#lines[@]}" -eq 7 ]
}

@test "this kernel flavor restart is OK" {
    run "$CMD" "gke-kernel-api-e2e-tests" "${TEST_FIXTURES}/kernel-collector-previous.log"
    [ "$status" -eq 0 ]
}

@test "this ebpf flavor restart is OK" {
    run "$CMD" "gke-api-e2e-tests" "${TEST_FIXTURES}/ebpf-collector-previous.log"
    [ "$status" -eq 0 ]
}

teardown () {
    echo "$BATS_TEST_NAME
--------
$output
--------

"
}
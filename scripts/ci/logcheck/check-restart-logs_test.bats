#!/usr/bin/env bats

CMD="${BATS_TEST_DIRNAME}/check-restart-logs.sh"
TEST_FIXTURES="${BATS_TEST_DIRNAME}/test_fixtures"

function setup() {
    if [[ -z "${ARTIFACT_DIR:-}" ]]; then
        export ARTIFACT_DIR="${BATS_FILE_TMPDIR}"
    fi
}

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
    [ "${lines[1]}" = "Ignoring this restart due to: collector restart due to sensor connection failure (likely slow start)" ]
}

@test "it can depend on process" {
    run "$CMD" "openshift-crio-api-e2e-tests" "${TEST_FIXTURES}/other-process-previous.log"
    [ "$status" -eq 2 ]
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
    run "$CMD" "openshift-api-e2e-tests" "${TEST_FIXTURES}/exception-collector-previous.log" "${TEST_FIXTURES}/slow-sensor-collector-previous.log"
    [ "$status" -eq 0 ]
}

@test "it spots a log with no exceptions with other logs that have an exception (by content)" {
    run "$CMD" "openshift-api-e2e-tests" "${TEST_FIXTURES}/no-exception-collector-previous.log" "${TEST_FIXTURES}/exception-collector-previous.log"
    [ "$status" -eq 2 ]
}

@test "it spots a log with no exceptions with other logs that have an exception (by process)" {
    run "$CMD" "openshift-api-e2e-tests" "${TEST_FIXTURES}/other-process-previous.log" "${TEST_FIXTURES}/exception-collector-previous.log"
    [ "$status" -eq 2 ]
}

@test "ordering is not a problem" {
    run "$CMD" "openshift-api-e2e-tests" "${TEST_FIXTURES}/exception-collector-previous.log" "${TEST_FIXTURES}/no-exception-collector-previous.log"
    [ "$status" -eq 2 ]
}

@test "checks them all" {
    run "$CMD" "openshift-api-e2e-tests" "${TEST_FIXTURES}/exception-collector-previous.log" "${TEST_FIXTURES}/no-exception-collector-previous.log"
    [ "$status" -eq 2 ]
    [ "${#lines[@]}" -eq 5 ]
}

@test "collector sensor connection failure is OK for any job" {
    run "$CMD" "gke-api-e2e-tests" "${TEST_FIXTURES}/exception-collector-previous.log"
    [ "$status" -eq 0 ]
}

@test "scanner restart during upgrade postgres bounce is OK" {
    run "$CMD" "gke-upgrade-tests-central" "${TEST_FIXTURES}/upgrade-scanner-previous.log"
    [ "$status" -eq 0 ]
    [ "${lines[1]}" = "Ignoring this restart due to: scanner restart due to DB connection loss during postgres bounce in upgrade test" ]
}

@test "scanner restart during upgrade postgres bounce is NOT OK for non-upgrade jobs" {
    run "$CMD" "gke-nongroovy-e2e-tests" "${TEST_FIXTURES}/upgrade-scanner-previous.log"
    [ "$status" -eq 2 ]
}

@test "Scanner V4 connection-refused panic is allowed for indexer in N-3 rollback smoke" {
    run "$CMD" "gke-upgrade-tests-central" "${TEST_FIXTURES}/05_rollback_smoke/stackrox/pods/scanner-v4-indexer-589968d89c-vnkmh-indexer-previous.log"
    [ "$status" -eq 0 ]
    [ "${lines[1]}" = "Ignoring this restart due to: Scanner V4 N-3 rollback smoke raced Scanner V4 DB startup" ]
}

@test "Scanner V4 connection-refused panic is allowed for matcher in N-3 rollback smoke" {
    run "$CMD" "gke-upgrade-tests-central" "${TEST_FIXTURES}/05_rollback_smoke/stackrox/pods/scanner-v4-matcher-589c467548-jb5qc-matcher-previous.log"
    [ "$status" -eq 0 ]
}

@test "Scanner V4 rollback-smoke exception is not allowed for another job" {
    run "$CMD" "gke-other-job" "${TEST_FIXTURES}/05_rollback_smoke/stackrox/pods/scanner-v4-indexer-589968d89c-vnkmh-indexer-previous.log"
    [ "$status" -eq 2 ]
}

@test "Scanner V4 rollback-smoke exception is not allowed in another phase" {
    local other_phase="${BATS_FILE_TMPDIR}/04_postgres_postgres_rollback/stackrox/pods/scanner-v4-indexer-589968d89c-vnkmh-indexer-previous.log"
    mkdir -p "$(dirname "$other_phase")"
    cp "${TEST_FIXTURES}/05_rollback_smoke/stackrox/pods/scanner-v4-indexer-589968d89c-vnkmh-indexer-previous.log" "$other_phase"

    run "$CMD" "gke-upgrade-tests-central" "$other_phase"
    [ "$status" -eq 2 ]
}

@test "Scanner V4 rollback-smoke exception does not allow other errors" {
    local other_error="${BATS_FILE_TMPDIR}/05_rollback_smoke/stackrox/pods/scanner-v4-indexer-589968d89c-vnkmh-indexer-previous.log"
    mkdir -p "$(dirname "$other_error")"
    cp "${TEST_FIXTURES}/05_rollback_smoke/stackrox/pods/scanner-v4-indexer-589968d89c-vnkmh-indexer-other-error-previous.log" "$other_error"

    run "$CMD" "gke-upgrade-tests-central" "$other_error"
    [ "$status" -eq 2 ]
}

@test "Scanner V4 rollback-smoke exception does not allow other containers" {
    run "$CMD" "gke-upgrade-tests-central" "${TEST_FIXTURES}/05_rollback_smoke/stackrox/pods/scanner-v4-db-589968d89c-vnkmh-db-previous.log"
    [ "$status" -eq 2 ]
}

teardown () {
    echo "$BATS_TEST_NAME
--------
$output
--------

"
}

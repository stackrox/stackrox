#!/usr/bin/env bats

load "../../../scripts/test_helpers.bats"

# Real previous-log names from collect-service-logs.sh: <pod>-<container>-previous.log.
# Matcher/indexer pods are scanner-v4-matcher-* / scanner-v4-indexer-*, not scanner-v4-*.
@test "POD_CONTAINERS_MAP matches scanner-v4-matcher and indexer previous logs" {
    source "${BATS_TEST_DIRNAME}/../lib.sh"

    local matcher_re indexer_re
    matcher_re="${POD_CONTAINERS_MAP["pod: scanner-v4-matcher - container: matcher"]}"
    indexer_re="${POD_CONTAINERS_MAP["pod: scanner-v4-indexer - container: indexer"]}"

    [[ -n "${matcher_re}" ]]
    [[ -n "${indexer_re}" ]]
    [[ "scanner-v4-matcher-b49db6cbf-kg6p8-matcher-previous.log" =~ ${matcher_re} ]]
    [[ "scanner-v4-indexer-65d57884b9-c8dt2-indexer-previous.log" =~ ${indexer_re} ]]
    # Combined scanner-v4-<rs>-<hash> names must not match (those pods do not exist).
    [[ ! "scanner-v4-111111111-11111-matcher-previous.log" =~ ${matcher_re} ]]
}

@test "junit_report_pod_restarts - clean" {
    source "${BATS_TEST_DIRNAME}/../lib.sh"

    save_junit_failure() {
        echo "Fail: $1 $2"
    }

    save_junit_success() {
        echo "Success: $*"
    }

    POD_CONTAINERS_MAP=()
    POD_CONTAINERS_MAP["pod: test"]="test-[A-Za-z0-9]+-[A-Za-z0-9]+-test-previous.log"

    run junit_report_pod_restarts
    assert_success

    [ "${#lines[@]}" -eq 1 ]
    assert_line 'Success: Check unexpected pod restarts pod: test'
}

@test "junit_report_pod_restarts - failures" {
    source "${BATS_TEST_DIRNAME}/../lib.sh"

    save_junit_failure() {
        echo "Fail: $1 $2"
    }

    save_junit_success() {
        echo "Success: $*"
    }

    POD_CONTAINERS_MAP=()

    # Deployment failure
    POD_CONTAINERS_MAP["pod: scanner-v4-matcher - container: matcher"]="scanner-v4-matcher-[A-Za-z0-9]+-[A-Za-z0-9]+-matcher-previous.log"
    # DaemonSet failure (with two pods failed)
    POD_CONTAINERS_MAP["pod: collector - container: node-inventory"]="collector-[A-Za-z0-9]+-node-inventory-previous.log"
    # No failure
    POD_CONTAINERS_MAP["pod: sensor - container: sensor"]="sensor-[A-Za-z0-9]+-[A-Za-z0-9]+-sensor-previous.log"

    run junit_report_pod_restarts "$(cat "${BATS_TEST_DIRNAME}/fixtures/check-restart-logs-output.txt")"
    assert_success

    # 1 without failure and 3 failed (4 in output, but 1 de-duplicated)
    [ "${#lines[@]}" -eq 4 ]
    assert_line 'Success: Check unexpected pod restarts pod: sensor - container: sensor'
    assert_line 'Fail: Check unexpected pod restarts pod: scanner-v4-matcher - container: matcher'
    assert_line 'Fail: Check unexpected pod restarts pod: collector - container: node-inventory'
    assert_line 'Fail: Check unexpected pod restarts unknown'
}

@test "junit_report_pod_restarts - failure includes log" {
    source "${BATS_TEST_DIRNAME}/../lib.sh"

    save_junit_failure() {
        echo "Fail: $1 $2 - Log: $3"
    }

    save_junit_success() {
        echo "Success: $*"
    }

    POD_CONTAINERS_MAP=()

    # Using $'' - to create new line.
    run junit_report_pod_restarts $'Line1\nunknown-pod-111111111-11111-container-previous.log copied to Artifacts'
    assert_success

    [ "${#lines[@]}" -eq 2 ]
    assert_line --index 0 'Fail: Check unexpected pod restarts unknown - Log: Line1'
    assert_line --index 1 'unknown-pod-111111111-11111-container-previous.log copied to Artifacts'
}

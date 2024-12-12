#!/usr/bin/env bats

load "../../../scripts/test_helpers.bats"

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
    POD_CONTAINERS_MAP["pod: scanner-v4 - container: matcher"]="scanner-v4-[A-Za-z0-9]+-[A-Za-z0-9]+-matcher-previous.log"
    # DaemonSet failure (with two pods failed)
    POD_CONTAINERS_MAP["pod: collector - container: node-inventory"]="collector-[A-Za-z0-9]+-node-inventory-previous.log"
    # No failure
    POD_CONTAINERS_MAP["pod: sensor - container: sensor"]="sensor-[A-Za-z0-9]+-[A-Za-z0-9]+-sensor-previous.log"

    run junit_report_pod_restarts "$(cat "${BATS_TEST_DIRNAME}/fixtures/check-restart-logs-output.txt")"
    assert_success

    # 1 without failure and 3 failed (4 in output, but 1 de-duplicated)
    [ "${#lines[@]}" -eq 4 ]
    assert_line 'Success: Check unexpected pod restarts pod: sensor - container: sensor'
    assert_line 'Fail: Check unexpected pod restarts pod: scanner-v4 - container: matcher'
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

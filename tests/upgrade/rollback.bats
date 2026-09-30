#!/usr/bin/env bats

setup() {
    set -o pipefail
    source "${BATS_TEST_DIRNAME}/rollback.sh"
    export IMAGE=registry/main:4.9.0
    export POD='{"items":[{"metadata":{"name":"central-test"},"spec":{"containers":[{"name":"central","image":"registry/main:4.9.0"}]},"status":{"conditions":[{"type":"Ready","status":"False"}],"containerStatuses":[{"name":"central","ready":false,"lastState":{"terminated":{"exitCode":1}}}]}}]}'
    export LOG='Software downgrade is not supported.  The software supports database version of 213 but the database requires the software support a database version to be at least least 220'
}

@test "new rollback diagnostic is recognized" {
    rollback_rejection_observed "$POD" 'Central rollback blocked: database ACS "5.1" requires sequence 220, but ACS 4.9.0 only supports sequence 213. Use a compatible Central version.' "$IMAGE" 213 220
}

@test "polling is bounded and retains evidence without real sleeps" {
    export REGISTRY=registry
    kubectl() {
        case "$*" in
            *'get pods'*) echo "$POD" ;;
            *logs*) echo 'connection refused' ;;
        esac
    }
    sleep() { echo tick >>"$BATS_TEST_TMPDIR/ticks"; }
    run wait_for_rollback_rejection 4.9.0 213 220 "$BATS_TEST_TMPDIR/evidence" 2
    [ "$status" -ne 0 ]
    [ "$(wc -l <"$BATS_TEST_TMPDIR/ticks")" -eq 2 ]
    [ -s "$BATS_TEST_TMPDIR/evidence/pods.json" ]
    [ -s "$BATS_TEST_TMPDIR/evidence/current.log" ]
}

@test "ready Central fails immediately even with rejection in previous logs" {
    export REGISTRY=registry
    kubectl() { echo "${POD//False/True}"; }
    sleep() { echo tick >>"$BATS_TEST_TMPDIR/ticks"; }
    run wait_for_rollback_rejection 4.9.0 213 220 "$BATS_TEST_TMPDIR/evidence"
    [ "$status" -ne 0 ]
    [[ "$output" == *'unexpectedly became ready'* ]]
    [ ! -f "$BATS_TEST_TMPDIR/ticks" ]
}

@test "rejected rollback verifies unchanged metadata and stops the rejected pod" {
    stop_rollback_central() { echo stop >>"$BATS_TEST_TMPDIR/calls"; }
    set_rollback_version() { echo "$1" >>"$BATS_TEST_TMPDIR/calls"; }
    wait_for_rollback_rejection() { echo reject >>"$BATS_TEST_TMPDIR/calls"; }
    rollback_version_snapshot() { echo '[{"minseqnum":220,"version":"5.1"}]'; }
    local plan='{"minimum_sequence":220,"rejected":{"tag":"4.9.0","sequence":213}}'
    test_rejected_rollback "$plan" "$BATS_TEST_TMPDIR/evidence"
    [ "$(cat "$BATS_TEST_TMPDIR/calls")" = $'stop\n4.9.0\nreject\nstop' ]
    cmp "$BATS_TEST_TMPDIR/evidence/version-before.json" "$BATS_TEST_TMPDIR/evidence/version-after.json"

    rollback_version_snapshot() {
        if [[ -f "$BATS_TEST_TMPDIR/queried" ]]; then
            echo '[{"minseqnum":220,"version":"4.9"}]'
        else
            touch "$BATS_TEST_TMPDIR/queried"
            echo '[{"minseqnum":220,"version":"5.1"}]'
        fi
    }
    run test_rejected_rollback "$plan" "$BATS_TEST_TMPDIR/evidence"
    [ "$status" -ne 0 ]
    [[ "$output" == *'changed the database version record'* ]]
}

@test "snapshot read failure never starts rollback" {
    stop_rollback_central() { :; }
    rollback_version_snapshot() { return 1; }
    set_rollback_version() { touch "$BATS_TEST_TMPDIR/started"; }
    run test_rejected_rollback '{"minimum_sequence":220}' "$BATS_TEST_TMPDIR/evidence"
    [ "$status" -ne 0 ]
    [ ! -f "$BATS_TEST_TMPDIR/started" ]
}

@test "recovery preserves failure and never touches database images" {
    export REGISTRY=registry CURRENT_TAG=5.1.x
    kubectl() { echo "$*" >>"$BATS_TEST_TMPDIR/calls"; return 1; }
    run recover_rollback_central 42 '{"data":{}}'
    [ "$status" -eq 42 ]
    [ "$(wc -l <"$BATS_TEST_TMPDIR/calls")" -eq 3 ]
    grep -F 'central=registry/main:5.1.x' "$BATS_TEST_TMPDIR/calls"
    ! grep -E 'central-db|scanner' "$BATS_TEST_TMPDIR/calls"
}

@test "release chart uses historical CLI but keeps current Central DB" {
    source "${BATS_TEST_DIRNAME}/lib.sh"
    export TEST_ROOT="$BATS_TEST_TMPDIR" TEST_HOST_PLATFORM=linux
    export CURRENT_TAG=5.1.x REGISTRY=registry
    info() { :; }
    wait_for_api() { :; }
    wait_for_central_db() { :; }
    wait_for_scanner_V4() { :; }
    old_roxctl() { echo "$*" >"$BATS_TEST_TMPDIR/roxctl-args"; }
    kubectl() {
        case "$*" in
            *'get secrets'*) echo '{"items":[{"metadata":{"name":"stackrox-generated-test","creationTimestamp":"2026-01-01"},"data":{"generated-values.yaml":"e30="}}]}' ;;
            *) return 1 ;;
        esac
    }
    helm() {
        case "$*" in
            *'upgrade --help'*) echo help ;;
            *upgrade*) cat >"$BATS_TEST_TMPDIR/values.yaml" ;;
        esac
    }
    upgrade_central_helm_to_head stackrox 4.10.0 registry old_roxctl 5.1.x
    grep -F 'helm output central-services' "$BATS_TEST_TMPDIR/roxctl-args"
    [ "$(yq e '.central.db.image.tag' "$BATS_TEST_TMPDIR/values.yaml")" = 5.1.x ]
    [ "$(yq e '.central.image.tag' "$BATS_TEST_TMPDIR/values.yaml")" = 4.10.0 ]
    [ "$(yq e '.scannerV4.db.image.tag' "$BATS_TEST_TMPDIR/values.yaml")" = 4.10.0 ]
}

@test "setting rollback preserves config and changes only Central" {
    export REGISTRY=registry
    kubectl() {
        case "$*" in
            *'get configmap/'*) echo '{"data":{"central-config.yaml":"maintenance:\n  safeMode: false\n"}}' ;;
            *'patch configmap/'*) printf '%s' "${!#}" >"$BATS_TEST_TMPDIR/patch.json" ;;
            *) echo "$*" >>"$BATS_TEST_TMPDIR/calls" ;;
        esac
    }
    set_rollback_version 4.10.0
    local config
    config="$(jq -r '.data["central-config.yaml"]' "$BATS_TEST_TMPDIR/patch.json")"
    [ "$(yq e '.maintenance.forceRollbackVersion' - <<<"$config")" = 4.10.0 ]
    [ "$(yq e '.maintenance.safeMode' - <<<"$config")" = false ]
    grep -F 'central=registry/main:4.10.0' "$BATS_TEST_TMPDIR/calls"
    ! grep -E 'central-db|scanner' "$BATS_TEST_TMPDIR/calls"
}

@test "rejection requires the expected image, nonzero exit and compatibility error" {
    rollback_rejection_observed "$POD" "$LOG" "$IMAGE" 213 220
}

@test "unrelated crash, missing logs and wrong sequences are not rejection" {
    ! rollback_rejection_observed "$POD" 'connection refused' "$IMAGE" 213 220
    ! rollback_rejection_observed "$POD" '' "$IMAGE" 213 220
    ! rollback_rejection_observed "$POD" "$LOG" "$IMAGE" 214 220
    ! rollback_rejection_observed "$POD" "$LOG" "$IMAGE" 213 221
}

@test "ready, wrong image, successful exit and absent pods are not rejection" {
    ! rollback_rejection_observed "${POD//False/True}" "$LOG" "$IMAGE" 213 220
    ! rollback_rejection_observed "$POD" "$LOG" other-image 213 220
    ! rollback_rejection_observed "${POD//exitCode\":1/exitCode\":0}" "$LOG" "$IMAGE" 213 220
    ! rollback_rejection_observed '{"items":[]}' "$LOG" "$IMAGE" 213 220
}

@test "current termination is accepted without requiring a restart" {
    rollback_rejection_observed "$(jq '.items[0].status.containerStatuses[0].state = .items[0].status.containerStatuses[0].lastState | del(.items[0].status.containerStatuses[0].lastState)' <<<"$POD")" "$LOG" "$IMAGE" 213 220
}

@test "version snapshot must contain exactly one record and the expected floor" {
    rollback_check_snapshot '[{"version":"5.1","minseqnum":220}]' 220
    ! rollback_check_snapshot '[]' 220
    ! rollback_check_snapshot '[{"minseqnum":220},{"minseqnum":220}]' 220
    ! rollback_check_snapshot '[{"minseqnum":213}]' 220
    ! rollback_check_snapshot 'invalid' 220
}

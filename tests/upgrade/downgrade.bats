#!/usr/bin/env bats

setup() {
    set -o pipefail
    unset CI CI_JOB_NAME
    source "${BATS_TEST_DIRNAME}/downgrade.sh"
    export IMAGE=registry/main:4.9.0
    export POD='{"items":[{"metadata":{"name":"central-test"},"spec":{"containers":[{"name":"central","image":"registry/main:4.9.0"}]},"status":{"conditions":[{"type":"Ready","status":"False"}],"containerStatuses":[{"name":"central","ready":false,"lastState":{"terminated":{"exitCode":1}}}]}}]}'
    export LOG='Software downgrade is not supported.  The software supports database version of 213 but the database requires the software support a database version to be at least least 220'
}

@test "new rollback diagnostic is recognized" {
    downgrade_rejection_observed "$POD" 'Central rollback blocked: database ACS "5.1" requires sequence 220, but ACS 4.9.0 only supports sequence 213. Use a compatible Central version.' "$IMAGE"
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
    run wait_for_downgrade_rejection 4.9.0 "$BATS_TEST_TMPDIR/evidence" 2
    [ "$status" -ne 0 ]
    [ "$(wc -l <"$BATS_TEST_TMPDIR/ticks")" -eq 2 ]
    [ -s "$BATS_TEST_TMPDIR/evidence/pods.json" ]
    [ -s "$BATS_TEST_TMPDIR/evidence/current.log" ]
}

@test "ready Central fails immediately even with rejection in previous logs" {
    export REGISTRY=registry
    kubectl() { echo "${POD//False/True}"; }
    sleep() { echo tick >>"$BATS_TEST_TMPDIR/ticks"; }
    run wait_for_downgrade_rejection 4.9.0 "$BATS_TEST_TMPDIR/evidence"
    [ "$status" -ne 0 ]
    [[ "$output" == *'unexpectedly became ready'* ]]
    [ ! -f "$BATS_TEST_TMPDIR/ticks" ]
}

@test "rejected rollback verifies unchanged metadata and stops the rejected pod" {
    stop_downgrade_central() { echo stop >>"$BATS_TEST_TMPDIR/calls"; }
    set_downgrade_version() { echo "$1" >>"$BATS_TEST_TMPDIR/calls"; }
    wait_for_downgrade_rejection() {
        [ "$1" = 4.9.0 ] && [ "$2" = "$BATS_TEST_TMPDIR/evidence" ] || return 1
        echo reject >>"$BATS_TEST_TMPDIR/calls"
    }
    downgrade_version_snapshot() { echo '[{"minseqnum":220,"version":"5.1"}]'; }
    test_rejected_downgrade 4.9.0 "$BATS_TEST_TMPDIR/evidence"
    [ "$(cat "$BATS_TEST_TMPDIR/calls")" = $'stop\n4.9.0\nreject\nstop' ]
    cmp "$BATS_TEST_TMPDIR/evidence/version-before.json" "$BATS_TEST_TMPDIR/evidence/version-after.json"

    downgrade_version_snapshot() {
        if [[ -f "$BATS_TEST_TMPDIR/queried" ]]; then
            echo '[{"minseqnum":220,"version":"4.9"}]'
        else
            touch "$BATS_TEST_TMPDIR/queried"
            echo '[{"minseqnum":220,"version":"5.1"}]'
        fi
    }
    run test_rejected_downgrade 4.9.0 "$BATS_TEST_TMPDIR/evidence"
    [ "$status" -ne 0 ]
    [[ "$output" == *'changed the database version record'* ]]
}

@test "snapshot read failure never starts rollback" {
    stop_downgrade_central() { :; }
    downgrade_version_snapshot() { return 1; }
    set_downgrade_version() { touch "$BATS_TEST_TMPDIR/started"; }
    run test_rejected_downgrade 4.9.0 "$BATS_TEST_TMPDIR/evidence"
    [ "$status" -ne 0 ]
    [ ! -f "$BATS_TEST_TMPDIR/started" ]
}

@test "snapshot authenticates through stdin without a mounted password file" {
    setup_snapshot_commands
    run downgrade_version_snapshot
    [ "$status" -eq 0 ]
    [ "$output" = '[{"minseqnum":220,"version":"5.1"}]' ]
    [ -f "$BATS_TEST_TMPDIR/authenticated" ]
    grep -Fx -- '-i' "$BATS_TEST_TMPDIR/kubectl-args"
    grep -Fx 'deploy/central-db' "$BATS_TEST_TMPDIR/kubectl-args"
    grep -Fx 'central-db' "$BATS_TEST_TMPDIR/kubectl-args"
    grep -Fx -- '--no-password' "$BATS_TEST_TMPDIR/psql-args"
    grep -Fx 'ON_ERROR_STOP=1' "$BATS_TEST_TMPDIR/psql-args"
    run grep -F -e "$SNAPSHOT_PASSWORD" -e "$SNAPSHOT_SECRET" "$BATS_TEST_TMPDIR/kubectl-args" "$BATS_TEST_TMPDIR/psql-args"
    [ "$status" -eq 1 ]
}

@test "snapshot rejects input, transport and SQL failures without caller pipefail" {
    local scenario
    for scenario in secret-read malformed-base64 empty-password remote sql empty-output malformed-output; do
        setup_snapshot_commands
        rm -f "$BATS_TEST_TMPDIR/exec-called" "$BATS_TEST_TMPDIR/authenticated"
        case "$scenario" in
            secret-read) export SNAPSHOT_GET_STATUS=1 ;;
            malformed-base64) export SNAPSHOT_SECRET="${SNAPSHOT_SECRET}!" ;;
            empty-password) export SNAPSHOT_SECRET='' ;;
            remote) export SNAPSHOT_EXEC_STATUS=1 ;;
            sql) export SNAPSHOT_SQL_STATUS=1 ;;
            empty-output) export SNAPSHOT_JSON='' ;;
            malformed-output) export SNAPSHOT_JSON=invalid ;;
        esac
        set +o pipefail
        run downgrade_version_snapshot
        [ "$status" -ne 0 ]
        case "$scenario" in
            secret-read|malformed-base64|empty-password) [ ! -f "$BATS_TEST_TMPDIR/exec-called" ] ;;
            sql|empty-output|malformed-output) [ -f "$BATS_TEST_TMPDIR/authenticated" ] ;;
        esac
    done
}

@test "snapshot hides credentials from tracing and preserves caller shell options" {
    setup_snapshot_commands
    traced_snapshot() {
        set +o pipefail
        set -x
        downgrade_version_snapshot
        local result=$?
        [[ "$-" == *x* ]] || return 99
        set +x
        [[ ! -o pipefail ]] || return 99
        return "$result"
    }
    run traced_snapshot
    [ "$status" -eq 0 ]
    [ -f "$BATS_TEST_TMPDIR/authenticated" ]
    [[ "$output" != *"$SNAPSHOT_PASSWORD"* ]]
    [[ "$output" != *"$SNAPSHOT_SECRET"* ]]
    [[ ! -o xtrace ]]
    [[ -o pipefail ]]

    export SNAPSHOT_SQL_STATUS=1
    run traced_snapshot
    [ "$status" -eq 1 ]
    [[ "$output" != *"$SNAPSHOT_PASSWORD"* ]]
    [[ "$output" != *"$SNAPSHOT_SECRET"* ]]
}

@test "recovery preserves failure and never touches database images" {
    export REGISTRY=registry CURRENT_TAG=5.1.x
    kubectl() { echo "$*" >>"$BATS_TEST_TMPDIR/calls"; return 1; }
    run recover_downgrade_central 42 '{"data":{}}'
    [ "$status" -eq 42 ]
    [ "$(wc -l <"$BATS_TEST_TMPDIR/calls")" -eq 3 ]
    grep -F 'central=registry/main:5.1.x' "$BATS_TEST_TMPDIR/calls"
    ! grep -E 'central-db|scanner' "$BATS_TEST_TMPDIR/calls"
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
    set_downgrade_version 4.10.0
    local config
    config="$(jq -r '.data["central-config.yaml"]' "$BATS_TEST_TMPDIR/patch.json")"
    [ "$(yq e '.maintenance.forceRollbackVersion' - <<<"$config")" = 4.10.0 ]
    [ "$(yq e '.maintenance.safeMode' - <<<"$config")" = false ]
    grep -F 'central=registry/main:4.10.0' "$BATS_TEST_TMPDIR/calls"
    ! grep -E 'central-db|scanner' "$BATS_TEST_TMPDIR/calls"
}

@test "rejection requires the expected image, nonzero exit and compatibility error" {
    downgrade_rejection_observed "$POD" "$LOG" "$IMAGE"
}

@test "unrelated crash and missing logs are not rejection" {
    ! downgrade_rejection_observed "$POD" 'connection refused' "$IMAGE"
    ! downgrade_rejection_observed "$POD" '' "$IMAGE"
}

@test "ready, wrong image, successful exit and absent pods are not rejection" {
    ! downgrade_rejection_observed "${POD//False/True}" "$LOG" "$IMAGE"
    ! downgrade_rejection_observed "$POD" "$LOG" other-image
    ! downgrade_rejection_observed "${POD//exitCode\":1/exitCode\":0}" "$LOG" "$IMAGE"
    ! downgrade_rejection_observed '{"items":[]}' "$LOG" "$IMAGE"
}

@test "current termination is accepted without requiring a restart" {
    downgrade_rejection_observed "$(jq '.items[0].status.containerStatuses[0].state = .items[0].status.containerStatuses[0].lastState | del(.items[0].status.containerStatuses[0].lastState)' <<<"$POD")" "$LOG" "$IMAGE"
}

@test "version snapshot must contain exactly one version record" {
    downgrade_check_snapshot '[{"version":"5.1","minseqnum":220}]'
    ! downgrade_check_snapshot '[]'
    ! downgrade_check_snapshot '[{"minseqnum":220},{"minseqnum":220}]'
    ! downgrade_check_snapshot '[{"minseqnum":213}]'
    ! downgrade_check_snapshot 'invalid'
}

setup_snapshot_commands() {
    export SNAPSHOT_PASSWORD=$'synthetic password \'"$\\end'
    export SNAPSHOT_GET_STATUS=0 SNAPSHOT_EXEC_STATUS=0 SNAPSHOT_SQL_STATUS=0
    SNAPSHOT_SECRET="$(printf '%s' "$SNAPSHOT_PASSWORD" | base64)"
    export SNAPSHOT_SECRET
    export SNAPSHOT_JSON='[ { "version": "5.1", "minseqnum": 220 } ]'
    mkdir -p "$BATS_TEST_TMPDIR/bin"
    export PATH="$BATS_TEST_TMPDIR/bin:$PATH"
    cat >"$BATS_TEST_TMPDIR/bin/psql" <<'EOF'
#!/bin/sh
printf '%s\n' "$@" >"$BATS_TEST_TMPDIR/psql-args"
[ "${PGPASSWORD:-}" = "$SNAPSHOT_PASSWORD" ] || exit 1
touch "$BATS_TEST_TMPDIR/authenticated"
printf '%s\n' "$SNAPSHOT_JSON"
exit "${SNAPSHOT_SQL_STATUS:-0}"
EOF
    chmod +x "$BATS_TEST_TMPDIR/bin/psql"
    kubectl() {
        printf '%s\n' "$@" >>"$BATS_TEST_TMPDIR/kubectl-args"
        case "$*" in
            '-n stackrox get secret central-db-password -o jsonpath={.data.password}')
                printf '%s' "$SNAPSHOT_SECRET"
                return "${SNAPSHOT_GET_STATUS:-0}"
                ;;
            '-n stackrox exec '*)
                touch "$BATS_TEST_TMPDIR/exec-called"
                if [[ "${SNAPSHOT_EXEC_STATUS:-0}" -ne 0 ]]; then
                    printf '%s\n' "$SNAPSHOT_JSON"
                    return "$SNAPSHOT_EXEC_STATUS"
                fi
                while [[ "$#" -gt 0 && "$1" != -- ]]; do shift; done
                [[ "$#" -gt 0 ]] || return 1
                shift
                "$@"
                ;;
            *) return 1 ;;
        esac
    }
}

@test "version based rejection is recognized" {
    downgrade_rejection_observed "$POD" 'Central rollback blocked: cannot downgrade database from ACS 5.5 to ACS 5.1; maximum supported database source for this destination is ACS 5.4.' "$IMAGE"
}

@test "fixture stream must match before deployment" {
    check_downgrade_target 5.1.x-12-gabcdef 5.1
    check_downgrade_target 5.1.0-rc.1 5.1
    ! check_downgrade_target 5.2.0 5.1
    ! check_downgrade_target invalid 5.1
}

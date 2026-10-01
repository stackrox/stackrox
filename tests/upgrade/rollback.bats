#!/usr/bin/env bats

setup() {
    set -o pipefail
    unset CI CI_JOB_NAME
    source "${BATS_TEST_DIRNAME}/rollback.sh"
    export IMAGE=registry/main:4.9.0
    export POD='{"items":[{"metadata":{"name":"central-test"},"spec":{"containers":[{"name":"central","image":"registry/main:4.9.0"}]},"status":{"conditions":[{"type":"Ready","status":"False"}],"containerStatuses":[{"name":"central","ready":false,"lastState":{"terminated":{"exitCode":1}}}]}}]}'
    export LOG='Software downgrade is not supported.  The software supports database version of 213 but the database requires the software support a database version to be at least least 220'
}

@test "CI DB defaults customization is scoped and idempotent" {
    source "${BATS_TEST_DIRNAME}/lib.sh"
    export TEST_ROOT="$(cd "$BATS_TEST_DIRNAME/../.." && pwd)"
    mkdir -p "$BATS_TEST_TMPDIR/chart/internal"
    local defaults="$BATS_TEST_TMPDIR/chart/internal/defaults.yaml"
    cat >"$defaults" <<'EOF'
defaults:
  central:
    endpoint: "central.{{ required "unknown namespace" .Release.Namespace }}.svc:443"
    db:
      resources:
        requests:
          memory: "8Gi"
          cpu: "4"
        limits:
          memory: "16Gi"
          cpu: "8"
EOF
    cp "$defaults" "$BATS_TEST_TMPDIR/original"
    customize_ci_central_db_chart "$BATS_TEST_TMPDIR/chart"
    cmp "$defaults" "$BATS_TEST_TMPDIR/original"
    export CI=true CI_JOB_NAME=gke-upgrade-tests-postgres
    customize_ci_central_db_chart "$BATS_TEST_TMPDIR/chart"
    cmp "$defaults" "$BATS_TEST_TMPDIR/original"
    export CI_JOB_NAME=gke-upgrade-tests-central
    customize_ci_central_db_chart "$BATS_TEST_TMPDIR/chart"
    sed '/endpoint:/d' "$defaults" | yq e -j '.defaults.central.db.resources' - | jq -e '. == {"requests": {"cpu": "4", "memory": "8Gi"}, "limits": {"memory": "16Gi"}}'
    cp "$defaults" "$BATS_TEST_TMPDIR/customized"
    customize_ci_central_db_chart "$BATS_TEST_TMPDIR/chart"
    cmp "$defaults" "$BATS_TEST_TMPDIR/customized"
    echo unexpected >"$defaults"
    run customize_ci_central_db_chart "$BATS_TEST_TMPDIR/chart"
    [ "$status" -ne 0 ]
    [ "$(cat "$defaults")" = unexpected ]
}

@test "CI upgrade rejects retained limits and failed value reads without leaking values" {
    source "${BATS_TEST_DIRNAME}/lib.sh"
    export CI=true CI_JOB_NAME=gke-upgrade-tests-central
    export CURRENT_TAG=5.1.x REGISTRY=registry TEST_ROOT="$BATS_TEST_TMPDIR" TEST_HOST_PLATFORM=linux
    info() { :; }
    helm() {
        case "$*" in
            *'get values'*) echo '{"secret":"do-not-print","central":{"db":{"resources":{"limits":{"cpu":"8"}}}}}' ;;
        esac
    }
    run upgrade_central_helm_to_head
    [ "$status" -ne 0 ]
    [[ "$output" == *'retained Central DB CPU limit'* ]]
    [[ "$output" != *'do-not-print'* ]]
    helm() {
        if [[ "$*" == *'get values'* ]]; then
            echo '{}'
            return 42
        fi
    }
    run upgrade_central_helm_to_head
    [ "$status" -ne 0 ]
    [[ "$output" == *'retained Central DB CPU limit'* ]]
    helm() { [[ "$*" != *'get values'* ]]; }
    run upgrade_central_helm_to_head
    [ "$status" -ne 0 ]
}

setup_historical_scale() {
    export MAIN_IMAGE_TAG=test
    source "${BATS_TEST_DIRNAME}/postgres_run.sh"
    mkdir -p "$BATS_TEST_TMPDIR/historical/scale"
    git -C "$TEST_ROOT" show "$EARLIER_SHA:scale/launch_workload.sh" >"$BATS_TEST_TMPDIR/original"
    cd "$BATS_TEST_TMPDIR/historical"
    git init -q
    cp "$BATS_TEST_TMPDIR/original" scale/launch_workload.sh
    chmod +x scale/launch_workload.sh
    export CI=true CI_JOB_NAME=gke-upgrade-tests-central
}

@test "historical scale patch changes only CI DB resources and restores on success and failure" {
    setup_historical_scale
    bash() {
        cp scale/launch_workload.sh "$BATS_TEST_TMPDIR/patched"
        return "${SCALE_STATUS:-0}"
    }
    run run_ci_scaled_workload
    [ "$status" -eq 0 ]
    cmp scale/launch_workload.sh "$BATS_TEST_TMPDIR/original"
    grep 'patch deploy/central-db' "$BATS_TEST_TMPDIR/patched" | head -1 | grep -F '"requests":{"memory":"8Gi","cpu":"2"},"limits":{"memory":"8Gi","cpu":null}'
    sed '/patch deploy\/central-db/d' "$BATS_TEST_TMPDIR/original" >"$BATS_TEST_TMPDIR/other-original"
    sed '/patch deploy\/central-db/d' "$BATS_TEST_TMPDIR/patched" >"$BATS_TEST_TMPDIR/other-patched"
    cmp "$BATS_TEST_TMPDIR/other-original" "$BATS_TEST_TMPDIR/other-patched"
    export SCALE_STATUS=42
    run run_ci_scaled_workload
    [ "$status" -eq 42 ]
    cmp scale/launch_workload.sh "$BATS_TEST_TMPDIR/original"
}

@test "historical scale patch mismatch prevents execution" {
    setup_historical_scale
    echo mismatch >scale/launch_workload.sh
    bash() { touch "$BATS_TEST_TMPDIR/started"; }
    run run_ci_scaled_workload
    [ "$status" -ne 0 ]
    [ ! -f "$BATS_TEST_TMPDIR/started" ]
    [ "$(cat scale/launch_workload.sh)" = mismatch ]
}

@test "historical scale emits the intended DB strategic merge patch" {
    setup_historical_scale
    mkdir -p scale/workloads scale/signatures
    touch scale/workloads/scale-test.yaml
    printf '#!/bin/sh\nexit 0\n' >scale/signatures/create-signature-integrations.sh
    chmod +x scale/signatures/create-signature-integrations.sh
    kubectl() {
        if [[ "$*" == 'get nodes -o json' ]]; then
            echo '{"items":[{},{},{},{}]}'
        elif [[ "$*" == *'patch deploy/central-db'* ]]; then
            printf '%s\n' "${!#}" >"$BATS_TEST_TMPDIR/db-patch"
        fi
    }
    export -f kubectl
    run_ci_scaled_workload
    jq -e '.spec.template.spec.containers == [{"name":"central-db","resources":{"requests":{"memory":"8Gi","cpu":"2"},"limits":{"memory":"8Gi","cpu":null}}}]' "$BATS_TEST_TMPDIR/db-patch"
    cmp scale/launch_workload.sh "$BATS_TEST_TMPDIR/original"
}

@test "historical scale is unchanged outside the Central CI job" {
    setup_historical_scale
    bash() { cmp scale/launch_workload.sh "$BATS_TEST_TMPDIR/original"; }
    export CI_JOB_NAME=gke-upgrade-tests-postgres
    run_ci_scaled_workload
    export CI_JOB_NAME=gke-upgrade-tests-central
    unset CI
    run_ci_scaled_workload
}

@test "initial install passes scoped DB request through Helm arguments" {
    source "${BATS_TEST_DIRNAME}/lib.sh"
    export CI=true CI_JOB_NAME=gke-upgrade-tests-central EARLIER_TAG=4.10.0 TEST_HOST_PLATFORM=linux
    info() { :; }
    go() { echo /tmp; }
    make() { :; }
    roxctl() { :; }
    gen_admin_password() { echo synthetic; }
    sleep() { :; }
    ci_export() { :; }
    customize_ci_central_db_chart() { echo "$1" >"$BATS_TEST_TMPDIR/customized"; }
    helm() { printf '%s\n' "$@" >"$BATS_TEST_TMPDIR/args"; }
    deploy_earlier_postgres_central
    grep -Fx /tmp/early-stackrox-central-services-chart "$BATS_TEST_TMPDIR/customized"
    grep -Fx -- --set-string "$BATS_TEST_TMPDIR/args"
    grep -Fx central.db.resources.requests.cpu=2 "$BATS_TEST_TMPDIR/args"
    grep -Fx central.db.resources.requests.memory=8Gi "$BATS_TEST_TMPDIR/args"
    grep -Fx central.db.resources.limits.memory=8Gi "$BATS_TEST_TMPDIR/args"
    rm "$BATS_TEST_TMPDIR/customized"
    export CI_JOB_NAME=gke-upgrade-tests-postgres
    deploy_earlier_postgres_central
    [ ! -e "$BATS_TEST_TMPDIR/customized" ]
    ! grep -F central.db.resources.requests.cpu "$BATS_TEST_TMPDIR/args"
}

@test "historical scale cleanup failures are reported and preserve command failure" {
    setup_historical_scale
    bash() { echo mismatch >scale/launch_workload.sh; return "${SCALE_STATUS:-0}"; }
    run run_ci_scaled_workload
    [ "$status" -ne 0 ]
    [[ "$output" == *'Failed to reverse Central DB CI scale patch'* ]]
    cp "$BATS_TEST_TMPDIR/original" scale/launch_workload.sh
    export SCALE_STATUS=42
    run run_ci_scaled_workload
    [ "$status" -eq 42 ]
    [[ "$output" == *'Failed to reverse Central DB CI scale patch'* ]]
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

@test "snapshot authenticates through stdin without a mounted password file" {
    setup_snapshot_commands
    run rollback_version_snapshot
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

@test "snapshot rejects failed secret lookup even when it produces valid data" {
    setup_snapshot_commands
    export SNAPSHOT_GET_STATUS=1
    set +o pipefail
    run rollback_version_snapshot
    [ "$status" -ne 0 ]
    [ ! -f "$BATS_TEST_TMPDIR/exec-called" ]
}

@test "snapshot rejects malformed base64 before database execution" {
    setup_snapshot_commands
    export SNAPSHOT_SECRET="${SNAPSHOT_SECRET}!"
    run rollback_version_snapshot
    [ "$status" -ne 0 ]
    [ ! -f "$BATS_TEST_TMPDIR/exec-called" ]
}

@test "snapshot rejects missing or empty password before database execution" {
    setup_snapshot_commands
    export SNAPSHOT_SECRET=''
    run rollback_version_snapshot
    [ "$status" -ne 0 ]
    [ ! -f "$BATS_TEST_TMPDIR/exec-called" ]
}

@test "snapshot propagates remote and SQL failures without caller pipefail" {
    setup_snapshot_commands
    set +o pipefail
    export SNAPSHOT_EXEC_STATUS=1
    run rollback_version_snapshot
    [ "$status" -ne 0 ]

    export SNAPSHOT_EXEC_STATUS=0 SNAPSHOT_SQL_STATUS=1
    run rollback_version_snapshot
    [ "$status" -ne 0 ]
    [ -f "$BATS_TEST_TMPDIR/authenticated" ]
}

@test "snapshot rejects empty and malformed query output" {
    setup_snapshot_commands
    local result
    for result in '' 'invalid'; do
        export SNAPSHOT_JSON="$result"
        run rollback_version_snapshot
        [ "$status" -ne 0 ]
        [ -f "$BATS_TEST_TMPDIR/authenticated" ]
    done
}

@test "snapshot hides credentials from tracing and preserves caller shell options" {
    setup_snapshot_commands
    traced_snapshot() {
        set +o pipefail
        set -x
        rollback_version_snapshot
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
    run recover_rollback_central 42 '{"data":{}}'
    [ "$status" -eq 42 ]
    [ "$(wc -l <"$BATS_TEST_TMPDIR/calls")" -eq 3 ]
    grep -F 'central=registry/main:5.1.x' "$BATS_TEST_TMPDIR/calls"
    ! grep -E 'central-db|scanner' "$BATS_TEST_TMPDIR/calls"
}

@test "release chart uses historical CLI but keeps current Central DB" {
    source "${BATS_TEST_DIRNAME}/lib.sh"
    export TEST_ROOT="$(cd "$BATS_TEST_DIRNAME/../.." && pwd)" TEST_HOST_PLATFORM=linux
    export CURRENT_TAG=5.1.x REGISTRY=registry
    info() { :; }
    wait_for_api() { :; }
    wait_for_central_db() { :; }
    wait_for_scanner_V4() { :; }
    old_roxctl() {
        echo "$*" >"$BATS_TEST_TMPDIR/roxctl-args"
        while [[ "$1" != --output-dir ]]; do shift; done
        mkdir -p "$2/internal"
        printf 'defaults:\n  central:\n    db:\n      resources:\n        requests:\n          memory: "8Gi"\n          cpu: "4"\n        limits:\n          memory: "16Gi"\n          cpu: "8"\n' >"$2/internal/defaults.yaml"
        echo "$2" >"$BATS_TEST_TMPDIR/chart-dir"
    }
    kubectl() {
        case "$*" in
            *'get secrets'*) echo '{"items":[{"metadata":{"name":"stackrox-generated-test","creationTimestamp":"2026-01-01"},"data":{"generated-values.yaml":"e30="}}]}' ;;
            *) return 1 ;;
        esac
    }
    helm() {
        case "$*" in
            *'upgrade --help'*) echo help ;;
            *'get values'*) echo '{"central":{"db":{"resources":{"limits":{"cpu":null}}}}}' ;;
            *upgrade*)
                printf '%s\n' "$@" >"$BATS_TEST_TMPDIR/helm-args"
                cat >"$BATS_TEST_TMPDIR/values.yaml"
                ;;
        esac
    }
    upgrade_central_helm_to_head stackrox 4.10.0 registry old_roxctl 5.1.x
    grep -F 'helm output central-services' "$BATS_TEST_TMPDIR/roxctl-args"
    [ "$(yq e '.central.db.image.tag' "$BATS_TEST_TMPDIR/values.yaml")" = 5.1.x ]
    [ "$(yq e '.central.image.tag' "$BATS_TEST_TMPDIR/values.yaml")" = 4.10.0 ]
    [ "$(yq e '.scannerV4.db.image.tag' "$BATS_TEST_TMPDIR/values.yaml")" = 4.10.0 ]
    ! grep -F central.db.resources.requests.cpu "$BATS_TEST_TMPDIR/helm-args"
    export CI=true CI_JOB_NAME=gke-upgrade-tests-central
    upgrade_central_helm_to_head stackrox 4.10.0 registry old_roxctl 5.1.x
    grep -Fx -- --reuse-values "$BATS_TEST_TMPDIR/helm-args"
    grep -Fx -- --set-string "$BATS_TEST_TMPDIR/helm-args"
    grep -Fx central.db.resources.requests.cpu=2 "$BATS_TEST_TMPDIR/helm-args"
    grep -Fx central.db.resources.requests.memory=8Gi "$BATS_TEST_TMPDIR/helm-args"
    grep -Fx central.db.resources.limits.memory=8Gi "$BATS_TEST_TMPDIR/helm-args"
    [ "$(yq e '.defaults.central.db.resources.limits.cpu' "$(cat "$BATS_TEST_TMPDIR/chart-dir")/internal/defaults.yaml")" = null ]
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

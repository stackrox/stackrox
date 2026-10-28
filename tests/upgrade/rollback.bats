#!/usr/bin/env bats

setup() {
    set -o pipefail
    unset CI CI_JOB_NAME
    source "${BATS_TEST_DIRNAME}/rollback.sh"
    export IMAGE=registry/main:4.9.0
    export POD='{"items":[{"metadata":{"name":"central-test"},"spec":{"containers":[{"name":"central","image":"registry/main:4.9.0"}]},"status":{"conditions":[{"type":"Ready","status":"False"}],"containerStatuses":[{"name":"central","ready":false,"lastState":{"terminated":{"exitCode":1}}}]}}]}'
    export LOG='Software downgrade is not supported.  The software supports database version of 213 but the database requires the software support a database version to be at least least 220'
}

@test "N-3 smoke cleanup preserves remote registration for current smoke" {
    export MAIN_IMAGE_TAG=test TEST_HOST_PLATFORM=linux API_ENDPOINT=central.example REGISTRY=registry
    export CLUSTER_TYPE_FOR_TEST=K8S EVENTS="$BATS_TEST_TMPDIR/events"
    export REMOTE_REGISTRATION="$BATS_TEST_TMPDIR/remote-registration"
    export RESOURCE_STATE="$BATS_TEST_TMPDIR/sensor-resources"
    mkdir -p "$RESOURCE_STATE"
    touch "$REMOTE_REGISTRATION"
    source "${BATS_TEST_DIRNAME}/postgres_run.sh"

    local n3_checkout="$BATS_TEST_TMPDIR/n3" current_checkout="$BATS_TEST_TMPDIR/current" checkout release
    for checkout in "$n3_checkout" "$current_checkout"; do
        if [[ "$checkout" == "$n3_checkout" ]]; then
            release=n3
        else
            release=current
        fi
        mkdir -p "$checkout/bin/linux"
        cat >"$checkout/bin/linux/roxctl" <<EOF
#!/usr/bin/env bash
if [[ "\$*" == *"sensor get-bundle remote"* ]]; then
    if [[ ! -f "\$REMOTE_REGISTRATION" ]]; then
        printf 'bundle-denied:$release\\n' >>"\$EVENTS"
        echo "Central remote registration missing for '$release' bundle" >&2
        exit 1
    fi
    printf 'bundle:$release\\n' >>"\$EVENTS"
    mkdir -p sensor-remote
    cat >sensor-remote/sensor.sh <<'SCRIPT'
#!/usr/bin/env bash
printf 'install:$release\\n' >>"\$EVENTS"
touch "\$RESOURCE_STATE/$release"
SCRIPT
    cat >sensor-remote/delete-sensor.sh <<'SCRIPT'
#!/usr/bin/env bash
printf 'delete-sensor:$release\\n' >>"\$EVENTS"
unlink "\$RESOURCE_STATE/$release"
SCRIPT
    chmod +x sensor-remote/sensor.sh sensor-remote/delete-sensor.sh
elif [[ "\$*" == *"cluster delete --name remote"* ]]; then
    printf 'cluster-delete:$release\\n' >>"\$EVENTS"
    unlink "\$REMOTE_REGISTRATION"
fi
EOF
        chmod +x "$checkout/bin/linux/roxctl"
    done

    info() { :; }
    kubectl() { :; }
    sensor_wait() { :; }
    wait_for_central_reconciliation() { :; }
    wait_for_scored_image_scan() { :; }
    remove_qa_test_results() { :; }
    store_qa_test_results() { :; }
    make() { :; }

    run_upgrade_smoke "$n3_checkout" 4.10.0 4.10.0 n3-results
    remove_rollback_smoke_sensor_resources "$n3_checkout"
    run_upgrade_smoke "$current_checkout" test test current-results

    [ "$(grep -c '^bundle:' "$EVENTS")" -eq 2 ]
    grep -Fx 'bundle:n3' "$EVENTS"
    grep -Fx 'install:n3' "$EVENTS"
    grep -Fx 'delete-sensor:n3' "$EVENTS"
    grep -Fx 'bundle:current' "$EVENTS"
    grep -Fx 'install:current' "$EVENTS"
    local n3_bundle n3_install cleanup current_bundle current_install
    n3_bundle="$(grep -nFx 'bundle:n3' "$EVENTS" | cut -d: -f1)"
    n3_install="$(grep -nFx 'install:n3' "$EVENTS" | cut -d: -f1)"
    cleanup="$(grep -nFx 'delete-sensor:n3' "$EVENTS" | cut -d: -f1)"
    current_bundle="$(grep -nFx 'bundle:current' "$EVENTS" | cut -d: -f1)"
    current_install="$(grep -nFx 'install:current' "$EVENTS" | cut -d: -f1)"
    [ "$n3_bundle" -lt "$cleanup" ]
    [ "$n3_install" -lt "$cleanup" ]
    [ "$cleanup" -lt "$current_bundle" ]
    [ "$current_bundle" -lt "$current_install" ]
    if grep -q '^cluster-delete:' "$EVENTS"; then
        false
    fi
    if grep -q '^delete-sensor:current$' "$EVENTS"; then
        false
    fi
    [ ! -e "$RESOURCE_STATE/n3" ]
    [ -e "$RESOURCE_STATE/current" ]
    [ -f "$REMOTE_REGISTRATION" ]
}

@test "initial install pins Scanner V4 images without DB resource overrides" {
    source "${BATS_TEST_DIRNAME}/lib.sh"
    export CI=true CI_JOB_NAME=gke-upgrade-tests-central EARLIER_TAG=4.10.0 CURRENT_TAG=5.1.x TEST_HOST_PLATFORM=linux
    info() { :; }
    go() { echo /tmp; }
    make() { :; }
    roxctl() { :; }
    gen_admin_password() { echo synthetic; }
    sleep() { return 1; }
    ci_export() { :; }
    helm() { printf '%s\n' "$@" >"$BATS_TEST_TMPDIR/args"; }
    deploy_earlier_postgres_central
    grep -Fx "scannerV4.image.tag=${EARLIER_TAG}" "$BATS_TEST_TMPDIR/args"
    grep -Fx "scannerV4.db.image.tag=${EARLIER_TAG}" "$BATS_TEST_TMPDIR/args"
    if grep -F 'central.db.resources' "$BATS_TEST_TMPDIR/args"; then
        false
    fi
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
        run rollback_version_snapshot
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

setup_release_chart() {
    source "${BATS_TEST_DIRNAME}/lib.sh"
    export TEST_ROOT="$BATS_TEST_TMPDIR" TEST_HOST_PLATFORM=linux
    export CURRENT_TAG=5.1.x REGISTRY=registry SCANNER_V4_DB_STORAGE_CLASS=faster
    info() { echo "$*"; }
    wait_for_api() { :; }
    wait_for_central_db() { :; }
    wait_for_scanner_V4() { :; }
    mock_roxctl() { echo "$*" >"$BATS_TEST_TMPDIR/roxctl-args"; }
    export -f mock_roxctl
    mkdir -p "$TEST_ROOT/bin/linux"
    printf '#!/usr/bin/env bash\nmock_roxctl "$@"\n' >"$TEST_ROOT/bin/linux/roxctl"
    chmod +x "$TEST_ROOT/bin/linux/roxctl"
    CHART_SECRETS="$(jq -n '{items: [
        {
            metadata: {name: "stackrox-generated-new", creationTimestamp: "2026-02-01"},
            data: {"generated-values.yaml": ("new CA values" | @base64)}
        },
        {
            metadata: {name: "stackrox-generated-old", creationTimestamp: "2026-01-01"},
            data: {"generated-values.yaml": ("old CA values" | @base64)}
        }
    ]}')"
    CHART_SECRET_STATUS=0
    kubectl() {
        echo "$*" >>"$BATS_TEST_TMPDIR/kubectl-args"
        [[ "$*" == *'get secrets'* ]] || return 1
        echo "$CHART_SECRETS"
        return "$CHART_SECRET_STATUS"
    }
    helm() {
        case "$*" in
            *'upgrade --help'*) echo help ;;
            *upgrade*)
                printf '%s\n' "$@" >"$BATS_TEST_TMPDIR/helm-args"
                cat >"$BATS_TEST_TMPDIR/values.yaml"
                while [[ "$1" != -f ]]; do
                    shift
                done
                cp "$2" "$BATS_TEST_TMPDIR/generated-values"
                ;;
        esac
    }
}

@test "release charts preserve current DB and reuse the oldest install-time secret" {
    setup_release_chart
    local tag
    for tag in 4.10.0 "$CURRENT_TAG"; do
        if [[ "$tag" == 4.10.0 ]]; then
            upgrade_central_helm "$tag" mock_roxctl
        else
            upgrade_central_helm
        fi
        grep -F 'helm output central-services' "$BATS_TEST_TMPDIR/roxctl-args"
        [ "$(yq e '.central.db.image.tag' "$BATS_TEST_TMPDIR/values.yaml")" = "$CURRENT_TAG" ]
        [ "$(yq e '.central.image.tag' "$BATS_TEST_TMPDIR/values.yaml")" = "$tag" ]
        [ "$(yq e '.scannerV4.db.image.tag' "$BATS_TEST_TMPDIR/values.yaml")" = "$tag" ]
        [ "$(cat "$BATS_TEST_TMPDIR/generated-values")" = "old CA values" ]
        grep -Fx -- --reuse-values "$BATS_TEST_TMPDIR/helm-args"
        run grep -F -e central.db.resources -e storageClass "$BATS_TEST_TMPDIR/helm-args"
        [ "$status" -eq 1 ]
        [ "$(yq e '.central.db.resources' "$BATS_TEST_TMPDIR/values.yaml")" = null ]
        [ "$(yq e '.scannerV4.db.persistence.persistentVolumeClaim.storageClass' "$BATS_TEST_TMPDIR/values.yaml")" = null ]
        # Exercise both API list orders; selection must depend on creation time.
        CHART_SECRETS="$(jq '.items |= reverse' <<<"$CHART_SECRETS")"
    done
    run grep -F 'apply' "$BATS_TEST_TMPDIR/kubectl-args"
    [ "$status" -eq 1 ]
}

@test "release chart rejects unusable generated values before Helm upgrade" {
    setup_release_chart
    local valid_secrets="$CHART_SECRETS" scenario
    for scenario in lookup-failure missing malformed-json invalid-base64 empty; do
        CHART_SECRETS="$valid_secrets"
        CHART_SECRET_STATUS=0
        case "$scenario" in
            lookup-failure)
                CHART_SECRET_STATUS=1
                ;;
            missing)
                CHART_SECRETS='{"items":[]}'
                ;;
            malformed-json)
                CHART_SECRETS=invalid
                ;;
            invalid-base64)
                CHART_SECRETS="$(jq '.items[].data["generated-values.yaml"] = "!"' <<<"$valid_secrets")"
                ;;
            empty)
                CHART_SECRETS="$(jq '.items[].data["generated-values.yaml"] = ""' <<<"$valid_secrets")"
                ;;
        esac
        rm -f "$BATS_TEST_TMPDIR/helm-args"
        run upgrade_central_helm
        [ "$status" -ne 0 ]
        [[ "$output" == *'Cannot read nonempty Helm generated values'* ]]
        [ ! -f "$BATS_TEST_TMPDIR/helm-args" ]
    done
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

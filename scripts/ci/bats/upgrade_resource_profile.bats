#!/usr/bin/env bats
# Bats isolates tests in subshells and invokes command mocks indirectly.
# shellcheck disable=SC1091,SC2030,SC2031,SC2329

setup() {
    TEST_ROOT="$(cd "${BATS_TEST_DIRNAME}/../../.." && pwd)"
    source "${TEST_ROOT}/tests/upgrade/lib.sh"
    unset UPGRADE_TEST_RESOURCE_PROFILE
    UPGRADE_RESOURCE_RENDERER="${BATS_TEST_TMPDIR}/renderer"
    touch "$UPGRADE_RESOURCE_RENDERER"
    helm() { echo "--post-renderer executable"; }
}

@test "resource profile is opt-in" {
    upgrade_resource_args central
    [ "${#UPGRADE_RESOURCE_ARGS[@]}" -eq 0 ]
}

@test "CI renderer is passed in historical checkout" {
    export UPGRADE_TEST_RESOURCE_PROFILE=ci
    cd "$BATS_TEST_TMPDIR"
    upgrade_resource_args central
    [ "${UPGRADE_RESOURCE_ARGS[0]}" = --post-renderer ]
    [ "${UPGRADE_RESOURCE_ARGS[1]}" = "$UPGRADE_RESOURCE_RENDERER" ]
    upgrade_resource_args secured-cluster
    [ "${UPGRADE_RESOURCE_ARGS[1]}" = "$UPGRADE_RESOURCE_RENDERER" ]
}

@test "unknown profile fails instead of silently using defaults" {
    export UPGRADE_TEST_RESOURCE_PROFILE=typo
    run upgrade_resource_args central
    [ "$status" -eq 1 ]
}

@test "Helm 4 receives isolated plugin name" {
    export UPGRADE_TEST_RESOURCE_PROFILE=ci
    helm() { echo 'postrenderer type plugin'; }
    upgrade_resource_args central
    [ "${UPGRADE_RESOURCE_ARGS[1]}" = ci-upgrade-resources ]
    [ -f "${HELM_PLUGINS}/ci-upgrade-resources/plugin.yaml" ]
}

@test "disabled profile retains exactly the two original patches" {
    kubectl() { printf '%s\n' "$*"; }
    run reconcile_upgrade_central_resources
    [ "$status" -eq 0 ]
    [ "${#lines[@]}" -eq 2 ]
    [ "${lines[0]}" = '-n stackrox patch deploy/central --type=strategic -p {"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"requests":{"cpu":"2"}}}]}}}}' ]
    [ "${lines[1]}" = '-n stackrox patch deploy/central-db --type=strategic -p {"spec":{"template":{"spec":{"containers":[{"name":"central-db","resources":{"requests":{"cpu":"2"}}}],"initContainers":[{"name":"init-db","resources":{"requests":{"cpu":"2"}}}]}}}}' ]
}

@test "single-node CI profile queries nodes without patching" {
    export UPGRADE_TEST_RESOURCE_PROFILE=ci
    kubectl() {
        printf '%s\n' "$*" >> "$BATS_TEST_TMPDIR/kubectl-calls"
        echo '{"items":[{}]}'
    }
    run reconcile_upgrade_central_resources
    [ "$status" -eq 0 ]
    [ "$(cat "$BATS_TEST_TMPDIR/kubectl-calls")" = 'get nodes -o json' ]
}

@test "multi-node CI profile fails without patching" {
    export UPGRADE_TEST_RESOURCE_PROFILE=ci
    kubectl() {
        printf '%s\n' "$*" >> "$BATS_TEST_TMPDIR/kubectl-calls"
        echo '{"items":[{},{}]}'
    }
    run reconcile_upgrade_central_resources
    [ "$status" -eq 1 ]
    [[ "$output" == *'requires exactly one node'* ]]
    [ "$(cat "$BATS_TEST_TMPDIR/kubectl-calls")" = 'get nodes -o json' ]
}

@test "node query failure propagates without patching" {
    export UPGRADE_TEST_RESOURCE_PROFILE=ci
    kubectl() { printf '%s\n' "$*" >> "$BATS_TEST_TMPDIR/kubectl-calls"; return 42; }
    run reconcile_upgrade_central_resources
    [ "$status" -ne 0 ]
    [ "$(cat "$BATS_TEST_TMPDIR/kubectl-calls")" = 'get nodes -o json' ]
}

@test "unknown profile fails before invoking kubectl" {
    export UPGRADE_TEST_RESOURCE_PROFILE=typo
    kubectl() { touch "$BATS_TEST_TMPDIR/unexpected-call"; }
    run reconcile_upgrade_central_resources
    [ "$status" -eq 1 ]
    [ ! -e "$BATS_TEST_TMPDIR/unexpected-call" ]
}

@test "renderer builds from TEST_ROOT rather than historical checkout" {
    export UPGRADE_TEST_RESOURCE_PROFILE=ci
    unset UPGRADE_RESOURCE_RENDERER
    go() { printf '%s\n%s\n' "$PWD" "$*" > "$BATS_TEST_TMPDIR/build"; }
    cd "$BATS_TEST_TMPDIR"
    upgrade_resource_args central
    [ "$(head -n 1 "$BATS_TEST_TMPDIR/build")" = "$TEST_ROOT" ]
    [[ "$(tail -n 1 "$BATS_TEST_TMPDIR/build")" == *'./tests/upgrade/resource-profile' ]]
}

#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Support for handling configurations where central is in one cluster and sensor
# is in another for e2e tests.

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$TEST_ROOT/scripts/lib.sh"
source "$TEST_ROOT/scripts/ci/lib.sh"

separate_cluster_test() {
    [[ "${SEPARATE_CLUSTER_TEST:-}" == "true" ]]
}

target_cluster() {
    if [[ "$#" -ne 1 ]]; then
        die "missing args. usage: target_cluster <central|sensor>"
    fi

    local target="$1"

    info "KUBE target is now $target"

    case "$target" in
        central)
            export KUBECONFIG="${CENTRAL_CLUSTER_KUBECONFIG}"
            ;;
        sensor)
            export KUBECONFIG="${SENSOR_CLUSTER_KUBECONFIG}"
            ;;
        *)
            die "a target of $target is not supported"
    esac
}

#!/usr/bin/env bash

# A collection of OpenShift related reusable bash functions for CI

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$SCRIPTS_ROOT/scripts/lib.sh"

set -euo pipefail

scale_worker_nodes() {
    info "Scaling worker nodes"

    if [[ "$#" -lt 1 ]]; then
        die "missing args. usage: scale_worker_nodes <node increment>"
    fi

    local increment="$1"

    info "Current node/machine state:"

    oc get nodes -o wide
    oc -n openshift-machine-api get machineset
    oc -n openshift-machine-api get machines

    original_machine_count="$(get_running_machine_count)"
    # (TODO(fixme): hack: rely on the machine sets all being workers)
    first_machine_set="$(oc -n openshift-machine-api get machineset -o json | jq -r '.items[0].metadata.name')"
    # (TODO(fixme): hack: relying on the machine set only having 1 running node)
    new_machineset_count=$((1 + increment))
    oc -n openshift-machine-api scale machineset "$first_machine_set" --replicas="$new_machineset_count"

    expected_machine_count=$((original_machine_count + increment))
    # note: timeout of this loop is handled by the calling context
    while [[ "$(get_running_machine_count)" != "$expected_machine_count" ]]; do
        info "Current machine state does not have the running count we desire ($expected_machine_count)"
        oc  -n openshift-machine-api get machines
        sleep 60
    done
}

get_running_machine_count() {
    oc -n openshift-machine-api get machines -o json | \
    jq -r '.items | map(select(.status.providerStatus.instanceState | ascii_upcase=="RUNNING")) | length'
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi

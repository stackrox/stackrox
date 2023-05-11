#!/usr/bin/env bash

# A collection of OpenShift related reusable bash functions for CI

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/lib.sh
source "$SCRIPTS_ROOT/scripts/lib.sh"

set -euo pipefail

scale_worker_nodes() {
    info "Scaling worker nodes"

    if [[ "$#" -lt 2 ]]; then
        die "missing args. usage: scale_worker_nodes <expected initial worker count> <node increment>"
    fi

    local expected_initial_worker_count="$1"
    local increment="$2"

    info "Current node/machine state:"

    oc get nodes -o wide
    oc -n openshift-machine-api get machineset
    oc -n openshift-machine-api get machines
    if [[ -n "${ARTIFACT_DIR:-}" ]]; then
        local scale_debug_dir="${ARTIFACT_DIR}/openshift/scaling-debug"
        mkdir -p "${scale_debug_dir}"
        oc -n openshift-machine-api get machineset -o json > "${scale_debug_dir}/initial_machineset.json"
        oc -n openshift-machine-api get machines -o json > "${scale_debug_dir}/initial_machines.json"
    fi

    # note: timeout of this loop is handled by the calling context
    while [[ "$(get_running_worker_count)" != "$expected_initial_worker_count" ]]; do
        info "Initial machine state does not have the running worker count we expect ($expected_initial_worker_count)"
        oc  -n openshift-machine-api get machines
        oc -n openshift-machine-api get machineset -o json > "${scale_debug_dir}/polled_initial_machineset.json"
        oc -n openshift-machine-api get machines -o json > "${scale_debug_dir}/polled_initial_machines.json"
        sleep 20
    done

    first_worker_machine_set="$(get_first_worker_machine_set)"
    current_replicas="$(oc -n openshift-machine-api get machineset "$first_worker_machine_set" -o json | jq -r '.spec.replicas')"
    new_machineset_count=$((current_replicas + increment))
    oc -n openshift-machine-api scale machineset "$first_worker_machine_set" --replicas="$new_machineset_count"

    expected_count=$((expected_initial_worker_count + increment))
    # note: timeout of this loop is handled by the calling context
    while [[ "$(get_running_worker_count)" != "$expected_count" ]]; do
        info "Current machine state does not have the running count we desire ($expected_count)"
        oc  -n openshift-machine-api get machines
        oc -n openshift-machine-api get machineset -o json > "${scale_debug_dir}/scaled_machineset.json"
        oc -n openshift-machine-api get machines -o json > "${scale_debug_dir}/scaled_machines.json"
        sleep 20
    done
}

get_running_worker_count() {
    oc -n openshift-machine-api get machines -l "machine.openshift.io/cluster-api-machine-role=worker" -o json | \
    jq -r '.items |
    map(select(
        (.status.providerStatus.instanceState | ascii_upcase)=="RUNNING"
           and
        (.status.phase | ascii_upcase)=="RUNNING"
        )) | length'
}

get_first_worker_machine_set() {
    oc -n openshift-machine-api get machineset -o json | \
    jq -r '[ .items[] |
             select(.spec.template.metadata.labels["machine.openshift.io/cluster-api-machine-role"]=="worker")
           ][0] | .metadata.name'
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi

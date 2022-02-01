#!/usr/bin/env bash

set -euo pipefail

# A collection of GKE related reusable bash functions for CI

set +u
SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
set -u

source "$SCRIPTS_ROOT/scripts/ci/lib.sh"

assign_env_variables() {
    info "Assigning environment variables for later steps"

    if [[ "$#" -lt 1 ]]; then
        die "missing args. usage: assign_env_variables <cluster-id> [<num-nodes> <machine-type>]"
    fi

    local cluster_id="$1"
    local num_nodes="${2:-3}"
    local machine_type="${3:-e2-standard-4}"

    ensure_CI

    if ! is_CIRCLECI; then
        die "Support is missing for this CI environment"
    fi

    require_environment "CIRCLE_BUILD_NUM"

    local cluster_name="rox-ci-${cluster_id}-${CIRCLE_BUILD_NUM}"
    ci_export CLUSTER_NAME "$cluster_name"
    echo "Assigned cluster name is $CLUSTER_NAME"

    ci_export NUM_NODES "$num_nodes"
    echo "Number of nodes for cluster is $NUM_NODES"

    ci_export MACHINE_TYPE "$machine_type"
    echo "Machine type is set as to $MACHINE_TYPE"

    local gke_release_channel="stable"
    if is_CIRCLECI; then
        if .circleci/pr_has_label.sh ci-gke-release-channel-rapid; then
            gke_release_channel="rapid"
        elif .circleci/pr_has_label.sh ci-gke-release-channel-regular; then
            gke_release_channel="regular"
        fi
    fi
    ci_export GKE_RELEASE_CHANNEL "$gke_release_channel"
    echo "Using gke release channel: $GKE_RELEASE_CHANNEL"
}

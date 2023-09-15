#!/usr/bin/env bash
set -eou pipefail

name=$1
flavor=$2
lifespan=$3
num_worker_nodes=${4:-3}

does_cluster_exist() {
    return "$(infractl get "$name" &> /dev/null; echo $?)"
}

if does_cluster_exist; then
    echo "A cluster with the name '${name}' already exists"
else
    echo "Creating an infra cluster with name '$name' and flavor '$flavor'"
    args=""
    if [[ "$flavor" == "openshift-4" ]]; then
        args="--arg worker-node-count=$num_worker_nodes"
    fi
    infractl create "$flavor" "$name" --description "Performance testing cluster" $args
    infractl lifespan "$name" "$lifespan"
fi

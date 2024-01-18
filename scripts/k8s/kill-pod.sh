#!/usr/bin/env bash

NAME=$1
NAMESPACE=${2:-stackrox}

mapfile -t pods < <(kubectl -n "${NAMESPACE}" get po --selector app="${NAME}" -o jsonpath='{.items[].metadata.name}')
if [[ ${#pods[@]} -gt 0 ]]; then
    kubectl -n "${NAMESPACE}" delete po "${pods[*]}" --grace-period=0
else
    echo "No pods to terminate"
fi

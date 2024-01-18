#! /bin/bash

NAME=$1
NAMESPACE=${2:-stackrox}

mapfile -t pods < <(kubectl -n "${NAMESPACE}" get po --selector app="${NAME}" -o jsonpath='{.items[].metadata.name}')
kubectl -n "${NAMESPACE}" delete po "${pods[*]}" --grace-period=0

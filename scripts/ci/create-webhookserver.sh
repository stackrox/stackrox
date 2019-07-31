#!/bin/bash

set -e

printstatus() {
    echo current resource status ...
    echo
    kubectl -n stackrox get all -l app=webhookserver
}

trap printstatus ERR

kubectl create -f webhookserver/server.yaml
sleep 5
POD=$(kubectl -n stackrox get pod -l app=webhookserver -o name)
kubectl -n stackrox wait --for=condition=ready "$POD" --timeout=2m
nohup kubectl -n stackrox port-forward "${POD}" 8080:8080 </dev/null > /dev/null 2>&1 &
sleep 1

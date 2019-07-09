#!/bin/bash

set -e

kubectl create -f webhookserver/server.yaml
sleep 5
POD=$(kubectl -n stackrox get pod -o jsonpath='{.items[?(@.metadata.labels.app=="webhookserver")].metadata.name}')
kubectl -n stackrox wait --for=condition=ready "pod/$POD" --timeout=2m
nohup kubectl -n stackrox port-forward "${POD}" 8080:8080 </dev/null > /dev/null 2>&1 &
sleep 1

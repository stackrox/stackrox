#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

if [[ -z "$1" ]]; then
  >&2 echo "usage: $0 <workload name> <num sensors>"
  exit 1
fi

if ! kubectl -n stackrox get deploy/central; then
  "$DIR"/launch_central.sh
  kubectl -n stackrox wait --for=condition=ready pod -l app=central --timeout 5m
else
  kubectl -n stackrox wait --for=condition=ready pod -l app=central --timeout 5m
  killpf 8000
  "$DIR"/port-forward.sh 8000
  sleep 5 # Allow port-forwards to initialize
fi
echo "Set retention settings"
roxcurl v1/config -X PUT -d @config.json

for i in $(seq 1 $2); do
  namespace="stackrox$i"
  kubectl get ns $namespace && kubectl delete ns $namespace
  kubectl create ns $namespace
  "$DIR"/launch_sensor.sh $1 $namespace
done

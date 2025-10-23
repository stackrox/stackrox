#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
export ROX_DIR="$( cd "${DIR}" && git rev-parse --show-toplevel)"

export LOCAL_PORT=${LOCAL_PORT:-8000}

git rev-parse --show-toplevel

if [[ -z "$1" ]]; then
  >&2 echo "usage: $0 <workload name> <num sensors>"
  exit 1
fi

if ! kubectl -n stackrox get deploy/central; then
  "$DIR"/launch_central.sh
  kubectl -n stackrox wait --for=condition=ready pod -l app=central --timeout 5m
else
  kubectl -n stackrox wait --for=condition=ready pod -l app=central --timeout 5m
  killpf "${LOCAL_PORT}"
  "$DIR"/port-forward.sh "${LOCAL_PORT}"
fi
echo "Set retention settings"
roxcurl v1/config -X PUT -d @config.json

for i in $(seq 1 $2); do
  namespace="stackrox$i"
  kubectl get ns $namespace && kubectl delete ns $namespace
  kubectl create ns $namespace
  "$DIR"/launch_sensor.sh $1 $namespace
done

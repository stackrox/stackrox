#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

if [[ -z "$1" ]]; then
  >&2 echo "usage: $0 <workload name> <num sensors>"
  exit 1
fi

workload_name=$1

if ! kubectl -n stackrox get deploy/central; then
  $DIR/launch_central.sh
fi

kubectl -n stackrox wait --for=condition=ready pod -l app=central --timeout 5m
killpf 8000
./port-forward.sh 8000
sleep 5 # Allow port-forwards to initialize

echo "Set retention settings"
roxcurl v1/config -X PUT -d @config.json

max_retry=3

for i in $(seq 1 $2); do
  namespace="stackrox$i"
  "$(launch_sensor_in_namespace "$workload_name" "$namespace" "$max_retry")"
done

function launch_sensor_in_namespace() {
  local workload_name=$1
  local namespace=$2
  local num_tries=$3

  if [[ "$num_tries" == 0 ]]; then
    return 0
  fi
  num_tries=((num_tries - 1))
  kubectl get ns $namespace && kubectl delete ns $namespace
  kubectl create ns $namespace
  ./launch_sensor.sh "$workload_name" "$namespace" || "$(launch_sensor_in_namespace "$workload_name" "$namespace" "$num_tries")"
}

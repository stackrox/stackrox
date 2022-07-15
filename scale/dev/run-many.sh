#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

if [[ -z "$1" ]]; then
  >&2 echo "usage: $0 <workload name> <num sensors>"
  exit 1
fi

if ! kubectl -n stackrox get deploy/central; then
  ./launch_central.sh
else
  killpf 8000
  ./port-forward.sh 8000
fi

for i in $(seq 1 $2); do
  namespace="stackrox$i"
  kubectl get ns $namespace && kubectl delete ns $namespace
  kubectl create ns $namespace
  ./launch_sensor.sh $1 $namespace
done

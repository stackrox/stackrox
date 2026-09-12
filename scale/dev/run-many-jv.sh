#!/bin/bash
set -x

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
STACKROX_DIR="$DIR/../.."

if [[ -z "$1" ]]; then
  >&2 echo "usage: $0 <workload name> <num sensors>"
  exit 1
fi

if ! kubectl -n stackrox get deploy/central; then
  "$DIR"/launch_central-jv.sh
  kubectl -n stackrox wait --for=condition=ready pod -l app=central --timeout 5m
else
  kubectl -n stackrox wait --for=condition=ready pod -l app=central --timeout 5m
  killpf 8000
  "$DIR"/port-forward-jv.sh 8000
  sleep 5 # Allow port-forwards to initialize
fi
#echo "Set retention settings"
#roxcurl v1/config -X PUT -d @config.json

start_time=$(date +%s)
"${STACKROX_DIR}/scratch/policies/disable-policies-db.sh"
end_time=$(date +%s)
duration=$((end_time - start_time))
echo "Disabled policies completed in ${duration} seconds."

for i in $(seq 1 $2); do
  namespace="stackrox$i"
  kubectl get ns $namespace && kubectl delete ns $namespace
  kubectl create ns $namespace
  "$DIR"/port-forward-jv.sh 8000
  "$DIR"/launch_sensor-jv.sh $1
  #"$DIR"/launch_sensor-jv.sh $1 $namespace
done

#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
export ROX_DIR="$( cd "${DIR}" && git rev-parse --show-toplevel)"

export LOCAL_PORT=${LOCAL_PORT:-8000}

if [[ -z "$1" ]]; then
  >&2 echo "usage: $0 <workload name> <num sensors>"
  exit 1
fi

if ! kubectl -n stackrox get deploy/central; then
  "$DIR"/launch_central.sh "${LOCAL_PORT}"
  kubectl -n stackrox wait --for=condition=ready pod -l app=central --timeout 5m
else
  kubectl -n stackrox wait --for=condition=ready pod -l app=central --timeout 5m
  killpf "${LOCAL_PORT}"
  "$DIR"/port-forward.sh "${LOCAL_PORT}"
fi
. .roxie.env
echo "Set retention settings"
roxcurl v1/config -X PUT -d @config.json

"$DIR"/kill-sensors.sh ${2:-1}
for i in $(seq 1 $2); do
  "$DIR"/launch_sensor.sh $1 "stackrox$i" || exit 1
done

#!/usr/bin/env bash
set -eu

# Wait for sensor to be up
#
#
#
# Usage:
#   sensor-wait.sh
#
# Example:
# $ ./scripts/ci/sensor-wait.sh
#

sensor_wait() {
    echo "Waiting for sensor to start"
    start_time="$(date '+%s')"
    while true; do
      sensor_json="$(kubectl -n stackrox get deploy/sensor -o json)"
      if [[ "$(jq '.status.replicas' <<<"${sensor_json}")" == 1 && "$(jq '.status.readyReplicas' <<<"${sensor_json}")" == 1 ]]; then
        break
      fi
      echo "Sensor replicas: $(jq '.status.replicas' <<<"${sensor_json}")"
      echo "Sensor readyReplicas: $(jq '.status.readyReplicas' <<<"${sensor_json}")"
      if (( $(date '+%s') - start_time > 1200 )); then
        kubectl -n stackrox get pod -o wide
        kubectl -n stackrox get deploy -o wide
        echo >&2 "Timed out after 20m"
        exit 1
      fi
      sleep 10
    done
    echo "Sensor is running"

    if ! kubectl -n stackrox get ds/collector ; then
        return
    fi

    echo "Waiting for collectors to start"
    start_time="$(date '+%s')"
    until [ "$(kubectl -n stackrox get ds/collector | tail -n +2 | awk '{print $2}')" -eq "$(kubectl -n stackrox get ds/collector | tail -n +2 | awk '{print $4}')" ]; do
      echo "Desired collectors: $(kubectl -n stackrox get ds/collector | tail -n +2 | awk '{print $2}')"
      echo "Ready collectors: $(kubectl -n stackrox get ds/collector | tail -n +2 | awk '{print $4}')"
      if (( $(date '+%s') - start_time > 1200 )); then
        kubectl -n stackrox get pod -o wide
        kubectl -n stackrox get ds -o wide
        echo >&2 "Timed out after 20m"
        exit 1
      fi
      sleep 10
    done
    echo "Collectors are running"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    sensor_wait "$@"
fi

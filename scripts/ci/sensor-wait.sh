#!/usr/bin/env bash
set -eu

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
ORCH_CMD=${ORCH_CMD:-"${ROOT}/scripts/retry-kubectl.sh"}

# Wait for sensor to be up
#
# Usage:
#   sensor-wait.sh [ <namespace> ]
#

sensor_wait() {
    local sensor_namespace=${1:-stackrox}

    echo "Waiting for sensor to start in namespace ${sensor_namespace}"

    start_time="$(date '+%s')"
    while true; do
      sensor_json="$("$ORCH_CMD" </dev/null -n "${sensor_namespace}" get deploy/sensor -o json)"
      if [[ "$(jq '.status.replicas' <<<"${sensor_json}")" == 1 && "$(jq '.status.readyReplicas' <<<"${sensor_json}")" == 1 ]]; then
        break
      fi
      echo "Sensor replicas: $(jq '.status.replicas' <<<"${sensor_json}")"
      echo "Sensor readyReplicas: $(jq '.status.readyReplicas' <<<"${sensor_json}")"
      if (( $(date '+%s') - start_time > 1200 )); then
        echo "Waiting for sensor in ns ${sensor_namespace} to be ready timed out." > "${QA_DEPLOY_WAIT_INFO}" || true
        "$ORCH_CMD" </dev/null -n "${sensor_namespace}" get pod -o wide
        "$ORCH_CMD" </dev/null -n "${sensor_namespace}" get deploy -o wide
        echo >&2 "Timed out after 20m"
        exit 1
      fi
      sleep 10
    done
    echo "Sensor is running"

    if ! "$ORCH_CMD" </dev/null -n "${sensor_namespace}" get ds/collector ; then
        return
    fi

    echo "Waiting for collectors to start"
    start_time="$(date '+%s')"
    until [ "$("$ORCH_CMD" </dev/null -n "${sensor_namespace}" get ds/collector | tail -n +2 | awk '{print $2}')" -eq "$("$ORCH_CMD" </dev/null -n "${sensor_namespace}" get ds/collector | tail -n +2 | awk '{print $4}')" ]; do
      echo "Desired collectors: $("$ORCH_CMD" </dev/null -n "${sensor_namespace}" get ds/collector | tail -n +2 | awk '{print $2}')"
      echo "Ready collectors: $("$ORCH_CMD" </dev/null -n "${sensor_namespace}" get ds/collector | tail -n +2 | awk '{print $4}')"
      if (( $(date '+%s') - start_time > 1200 )); then
        echo "Waiting for collectors in ns ${sensor_namespace} to be ready timed out." > "${QA_DEPLOY_WAIT_INFO}" || true
        "$ORCH_CMD" </dev/null -n "${sensor_namespace}" get pod -o wide
        "$ORCH_CMD" </dev/null -n "${sensor_namespace}" get ds -o wide
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

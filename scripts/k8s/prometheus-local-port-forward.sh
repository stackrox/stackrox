#!/usr/bin/env bash
set -e

LOCAL_PORT=${LOCAL_PORT:-9090}
NAMESPACE="${NAMESPACE:-stackrox}"

CENTRAL_POD=""
echo -n "Waiting for a running Central pod"
until [ -n "${CENTRAL_POD}" ]; do
  echo -n '.'
  sleep 2
  CENTRAL_POD="$(kubectl get pod -n "$NAMESPACE" --selector 'app=central' | grep Running | awk '{print $1}')"
done
echo "found Central pod: ${CENTRAL_POD}"

echo -n "Setting up Prometheus port-forwarding..."

# kubectl should be killed whenever this script is killed
trap 'kill -TERM ${PID}; wait ${PID}' TERM INT
kubectl port-forward -n "$NAMESPACE" "$CENTRAL_POD" "${LOCAL_PORT}:9090" > /dev/null &
PID=$!

until curl --output /dev/null --silent --fail -k "http://localhost:${LOCAL_PORT}/metrics"; do
    echo -n '.'
    sleep 1
done
echo "done"

echo "StackRox Central metrics are available on http://localhost:${LOCAL_PORT}/metrics"

wait ${PID}

#!/usr/bin/env bash
set -e

LOCAL_PORT=${LOCAL_PORT:-8000}
NAMESPACE="${NAMESPACE:-stackrox}"

pkill -f "kubectl port-forward -n ${NAMESPACE}" || true

CENTRAL_POD=""
echo -n "Waiting for a running Central pod"
until [ -n "${CENTRAL_POD}" ]; do
  echo -n '.'
  sleep 2
  CENTRAL_POD="$(kubectl get pod -n $NAMESPACE --selector 'app=central' | grep Running | awk '{print $1}')"
done
echo "found Central pod: ${CENTRAL_POD}"


echo -n "Setting up port-forwarding..."

# kubectl should be killed whenever this script is killed
trap 'kill -TERM ${PID}; wait ${PID}' TERM INT
kubectl port-forward -n "$NAMESPACE" "$CENTRAL_POD" "${LOCAL_PORT}:443" > /dev/null &
PID=$!

until $(curl --output /dev/null --silent --fail -k "https://localhost:${LOCAL_PORT}/v1/ping"); do
    echo -n '.'
    sleep 1
done
echo "done"

echo "StackRox Central is available on https://localhost:${LOCAL_PORT}/"

wait ${PID}

#!/usr/bin/env bash
set -e

LOCAL_PORT=${LOCAL_PORT:-8000}
NAMESPACE="${NAMESPACE:-stackrox}"

pkill -f "kubectl port-forward -n ${NAMESPACE}" || true
export CENTRAL_POD="$(kubectl get pod -n $NAMESPACE --selector 'app=central' --output=jsonpath='{.items..metadata.name} {.items..status.phase}' | grep Running | cut -f 1 -d ' ')"

echo "Setting port-forwarding..."

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

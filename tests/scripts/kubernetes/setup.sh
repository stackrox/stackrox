#!/usr/bin/env bash
set -e

export NAMESPACE="${NAMESPACE:-stackrox}"

pkill -f "kubectl port-forward -n ${NAMESPACE}" || true
export CENTRAL_POD="$(kubectl get pod -n $NAMESPACE --selector 'app=central' --output=jsonpath='{.items..metadata.name} {.items..status.phase}' | grep Running | cut -f 1 -d ' ')"
kubectl port-forward -n "$NAMESPACE" "$CENTRAL_POD" 8000:443 > /dev/null &

export LOCAL_API_ENDPOINT=localhost:8000
echo "Set local API endpoint to: $LOCAL_API_ENDPOINT"

until $(curl --output /dev/null --silent --fail -k "https://$LOCAL_API_ENDPOINT/v1/ping"); do
    echo -n '.'
    sleep 1
done

echo ""

while pgrep -f "^go test" > /dev/null; do sleep 1; done && pkill -f "kubectl port-forward " &
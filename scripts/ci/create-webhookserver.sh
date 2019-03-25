#! /bin/bash

CMD="$1"

if [[ -z $CMD ]]; then
    >&2 echo "First argument must be command name (kubectl or oc)"
    exit 1
fi

$CMD create -f webhookserver/server.yaml
sleep 5
POD=$($CMD -n stackrox get pod -o jsonpath='{.items[?(@.metadata.labels.app=="webhookserver")].metadata.name}')
$CMD  -n stackrox wait --for=condition=ready "pod/$POD"
$CMD  -n stackrox port-forward "${POD}" 8080:8080 > /dev/null &

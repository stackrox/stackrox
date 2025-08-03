#!/bin/sh

# Port Forward to StackRox Central
#
# Usage:
#   ./port-forward.sh <port> [namespace]
#
# Using a different command:
#     The KUBE_COMMAND environment variable will override the default of kubectl
#
# Examples:
# To use the default command to create resources:
#     $ ./port-forward.sh 8443
# To use another command instead:
#     $ export KUBE_COMMAND='kubectl --context prod-cluster'
#     $ ./port-forward.sh 8443

set -x

port="$1"
namespace=${2:-stackrox}

if [ -z "$1" ]; then
	echo "usage: $0 <port> [namespace]"
	echo "The above would forward localhost:<port> to central:443."
	exit 1
fi

while true; do
    central_pod="$(kubectl get pod -n "${namespace}" --selector 'app=central' --field-selector 'status.phase=Running' --output 'jsonpath={.items..metadata.name}' 2>/dev/null)"
    [ -z  "${central_pod}" ] || break
    printf '.'
    sleep 1
done
echo

max_attempts=300
count=1

nohup kubectl port-forward -n "${namespace}" svc/central "${port}:443" 1>/dev/null 2>&1 &
until nc -z 127.0.0.1 "$port"; do
    if [ "$count" -ge "$max_attempts" ]; then
        echo "Port $port did not become available after $max_attempts attempts. Exiting."
	exit 1
    fi
    echo "Waiting for port forward. Attempt $count of $max_attempts"
    sleep 1
    count=$((count + 1))
done
echo "Access central on https://localhost:${port}"

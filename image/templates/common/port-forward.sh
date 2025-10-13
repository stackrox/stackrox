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

if [ -z "$1" ]; then
	echo "usage: $0 <port> [namespace]"
	echo "The above would forward localhost:<port> to central:443."
	exit 1
fi

KUBE_COMMAND=${KUBE_COMMAND:-{{.K8sConfig.Command}}}
PORT="$1"
NAMESPACE=${2:-stackrox}

while true; do
    central_pod="$(${KUBE_COMMAND} get pod -n "${NAMESPACE}" --selector 'app=central' --field-selector 'status.phase=Running' --output 'jsonpath={.items..metadata.name}' 2>/dev/null)"
    [ -z  "${central_pod}" ] || break
    printf '.'
    sleep 1
done
echo

nohup "${KUBE_COMMAND}" port-forward -n "${NAMESPACE}" svc/central "${PORT}:443" --address='0.0.0.0' 1>/dev/null 2>&1 &
echo "Access central on https://localhost:${PORT}"

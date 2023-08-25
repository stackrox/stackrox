#!/bin/sh

# Port Forward to StackRox Central
#
# Usage:
#   ./port-forward.sh 8443
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

if [ -z "$1" ]; then
	echo "usage: $0 8443"
	echo "The above would forward localhost:8443 to central:443."
	exit 1
fi

while true; do
    central_pod="$(kubectl get pod -n 'stackrox' --selector 'app=central' --field-selector 'status.phase=Running' --output 'jsonpath={.items..metadata.name}' 2>/dev/null)"
    [ -z  "$central_pod" ] || break
    printf '.'
    sleep 1
done
echo

nohup kubectl port-forward -n 'stackrox' svc/central "$1:443" 1>/dev/null 2>&1 &
echo "Access central on https://localhost:$1"

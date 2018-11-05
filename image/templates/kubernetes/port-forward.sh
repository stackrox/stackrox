#!/bin/sh

if [ -z "$1" ]; then
	echo "usage: $0 8443"
	echo "The above would forward localhost:8443 to central:443."
	exit 1
fi

printf 'Connecting...'
while true; do
    central_pod="$(kubectl get pod -n '{{.K8sConfig.Namespace}}' --selector 'app=central' --field-selector 'status.phase=Running' --output 'jsonpath={.items..metadata.name}' 2>/dev/null)"
    [ -z  "$central_pod" ] || break
    printf '.'
    sleep 1
done
echo

nohup kubectl port-forward -n '{{.K8sConfig.Namespace}}' "$central_pod" "$1:443" 1>/dev/null 2>&1 &
echo "Access central on https://localhost:$1"

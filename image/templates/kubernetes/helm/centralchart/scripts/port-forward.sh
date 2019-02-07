#!/bin/sh

if [ -z "$1" ]; then
	echo "usage: $0 8443"
	echo "The above would forward localhost:8443 to central:443."
	exit 1
fi

printf 'Connecting...'
while true; do
    central_pod="$({{.K8sConfig.Command}} get pod -n 'stackrox' --selector 'app=central' --field-selector 'status.phase=Running' --output 'jsonpath={.items..metadata.name}' 2>/dev/null)"
    [ -z  "$central_pod" ] || break
    printf '.'
    sleep 1
done
echo

nohup {{.K8sConfig.Command}} port-forward -n 'stackrox' "$central_pod" "$1:443" 1>/dev/null 2>&1 &
echo "Access central on https://localhost:$1"

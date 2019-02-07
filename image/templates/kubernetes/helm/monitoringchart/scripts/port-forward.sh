#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

if [[ -z "$1" ]]; then
	echo "usage: bash monitoring-port-forward.sh 8443"
	echo "The above would forward localhost:8443 to monitoring:8443."
	exit 1
fi

until [ "$({{.K8sConfig.Command}} get pod -n stackrox --selector 'app=monitoring' | grep -c Running)" -eq 1 ]; do
    echo -n .
    sleep 1
done

export MONITORING_POD="$({{.K8sConfig.Command}} get pod -n stackrox --selector 'app=monitoring' --output=jsonpath='{.items..metadata.name} {.items..status.phase}' | grep Running | cut -f 1 -d ' ')"
{{.K8sConfig.Command}} port-forward -n "stackrox" "${MONITORING_POD}" "$1:8443" > /dev/null &
echo "Access monitoring on localhost:$1"

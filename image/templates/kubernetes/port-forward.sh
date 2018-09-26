#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

if [[ -z "$1" ]]; then
	echo "usage: bash port-forward.sh 8443"
	echo "The above would forward localhost:8443 to central:443."
	exit 1
fi

until [ "$(kubectl get pod -n {{.K8sConfig.Namespace}} --selector 'app=central' | grep Running | wc -l)" -eq 1 ]; do
    echo -n .
    sleep 1
done

export CENTRAL_POD="$(kubectl get pod -n {{.K8sConfig.Namespace}} --selector 'app=central' --output=jsonpath='{.items..metadata.name} {.items..status.phase}' | grep Running | cut -f 1 -d ' ')"
kubectl port-forward -n "{{.K8sConfig.Namespace}}" "${CENTRAL_POD}" "$1:443" > /dev/null &
echo "Access central on localhost:$1"

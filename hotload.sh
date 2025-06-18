#!/bin/bash

set -xeo pipefail

component="$1"
namespace="$2"

if [[ -z "$1" ]]; then
    echo "Usage: $0 [component] [namespace]"
    exit 0
fi

if [[ "$component" == "sensor" ]]; then
    binary_name=kubernetes
elif [[ "$component" == "central" ]]; then
    binary_name=central
else
    echo Provide component: sensor, central
    exit 1
fi

if [[ -z "$namespace" ]]; then
    echo "note: assuming default namespace"
    namespace="default"
fi

make bin/$binary_name
pod_name=$(kubectl -n "$namespace" get pod -l app=$component -oname)
hotload_cmd="cat - > /tmp/$binary_name && chmod +x /tmp/$binary_name && mv /tmp/$binary_name /stackrox && pkill $binary_name" 
kubectl exec -n "$namespace" -i $pod_name -- sh -c "$hotload_cmd" < bin/$binary_name

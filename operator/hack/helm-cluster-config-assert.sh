#!/bin/bash

set -eu -o pipefail

NAMESPACE=${NAMESPACE:-}
if [[ -z "$NAMESPACE" ]]; then
    echo >&2 "NAMESPACE must be set."
    exit 1
fi

config_yaml=$(\
    retry-kubectl.sh </dev/null -n "$NAMESPACE" get secret helm-cluster-config -o jsonpath='{.data.config\.yaml}' \
    | base64 --decode \
    | yq eval .clusterConfig - \
)

assertConfigValue() {
    config_path="$1"
    expected="$2"
    config_value=$(echo "$config_yaml" | yq eval "$config_path" -)
    [[ "$config_value" == "$expected" ]] || {
        echo "Assertion failed: ${config_path} != ${expected} (actual: $config_value)"
        return 1
    }
    return 0
}

if [[ $# != 2 ]]; then
    echo >&2 "$0: <property path> <property value>"
    exit 1
fi

assertConfigValue "$1" "$2"

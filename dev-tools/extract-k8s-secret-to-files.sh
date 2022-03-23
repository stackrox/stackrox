#!/usr/bin/env bash
set -eo pipefail

# Extract secret data entries into files on your local filesystem. Uses the default kubectl configuration.
# Usage: ./extract-k8s-secrets-to-files.sh [secret name] <namespace>
secret_name="$1"
namespace="$2"

if [[ "$#" -lt 1 ]]; then
    echo "Expected at least 1 arg, but found $#"
    echo "Usage: $0 [secret name] <namespace>"
    exit 1
fi

kubectl_args=()
if [[ -n "$namespace" ]]; then
  kubectl_args+=(
    -n "$namespace"
  )
fi

certs=$(kubectl "${kubectl_args[@]}" get secrets "$secret_name" -o json | jq -r '.data | keys[]')

for item in $(echo "$certs" | tr "\n" "\t");
do
  content=$(kubectl "${kubectl_args[@]}" get secrets "$secret_name" -o json | jq -r --arg key "$item" '.data[$key]')
  echo "$content" | base64 --decode > "$item"
  echo "Written data to $item"
done

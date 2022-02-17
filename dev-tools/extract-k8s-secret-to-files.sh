#!/usr/bin/env bash
set -eo pipefail

# Extract secret data entries into files on your local filesystem.
# Usage: ./extract-k8s-secrets-to-files.sh [namespace] [secret name]
namespace="$1"
secret_name="$2"

if [[ "$#" -ne 2 ]]; then
    echo "Expected 2 args, but found $#"
    echo "Usage: $0 [namespace] [secret name]"
    exit 1
fi

certs=$(kubectl -n "$namespace" get secrets "$secret_name" -o json | jq -r '.data | keys[]')

for item in $(echo "$certs" | tr "\n" "\t");
do
  echo "Written data to $item"
  content=$(kubectl -n "$namespace" get secrets "$secret_name" -o json | jq -r --arg key "$item" '.data[$key]')
  echo "$content" | base64 --decode > "$item"
done

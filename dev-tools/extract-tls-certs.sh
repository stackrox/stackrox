#!/usr/bin/env bash
set -eo pipefail

certs=$(kubectl -n stackrox get secrets sensor-tls -o json | jq -r '.data | keys[]')

for item in $(echo "$certs" | tr "\n" "\t");
do
  echo $item
  content=$(kubectl -n stackrox get secrets sensor-tls -o json | jq -r --arg key "$item" '.data[$key]')
  echo "$content" | base64 --decode > "$item"
done

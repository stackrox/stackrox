#!/bin/bash
set -euo pipefail

# Static values for distroless
originalValue="-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEWZzVzkb8A+DbgDpaJId/bOmV8n7Q\nOqxYbK0Iro6GzSmOzxkn+N2AKawLyXi84WSwJQBK//psATakCgAQKkNTAA==\n-----END PUBLIC KEY-----"
modifiedValue="-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE04soAoNygRhaytCtygPcwsP+6Ein\nYoDv/BJx1T9WmtsANh2HplRR66Fbm+3OjFuah2IhFufPhDl6a85I3ymVYw==\n-----END PUBLIC KEY-----"

tmp=$(mktemp)
status=$(curl -k -u "admin:${ROX_PASSWORD}" \
  -w "%{http_code}\n" \
  -o "$tmp" \
  https://central:443/v1/signatureintegrations )
if [ "${status}" != "200" ]; then
  cat "$tmp"
  exit 1
fi

integrationJSON=$(jq -c -r '.integrations[] | select( .name == "Distroless" )' "$tmp")
integrationID=$(echo "$integrationJSON" | jq -c -r '.id')

currentPublicKey=$(echo "$integrationJSON" | jq -c '.cosign.publicKeys[0].publicKeyPemEnc')

if [ "$currentPublicKey" = "\"$originalValue\"" ]; then
  currentPublicKey=$modifiedValue
else
  currentPublicKey=$originalValue
fi

replacedIntegrationJSON=$(echo "$integrationJSON" | jq -c -r ".cosign.publicKeys[0].publicKeyPemEnc = \"${currentPublicKey}\"")

tmp=$(mktemp)
status=$(curl -k -u "admin:${ROX_PASSWORD}" -X PUT \
  -d "${replacedIntegrationJSON}" \
  -o "$tmp" \
  -w "%{http_code}\n" \
  https://central:443/v1/signatureintegrations/"${integrationID}" )

if [ "${status}" != "200" ]; then
  cat "$tmp"
  exit 1
fi

exit 0

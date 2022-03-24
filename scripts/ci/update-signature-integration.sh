#!/usr/bin/env bash
set -x

# This script continuously updates the public key information of a signature integration to trigger reprocessing.
# This will be done in a 15 minute interval.

# Get the initial value of the first signature integration.
die() {
  echo >&2 "$@"
  exit 1
}

[[ -z "${ROX_PASSWORD}" ]] && die "Required env variable ROX_PASSWORD not set"
roxEndpoint="${API_ENDPOINT:-localhost:8000}"
roxUser="${ROX_ADMIN_USER:-admin}"
roxPassword="${ROX_PASSWORD}"

tmpOutput=$(mktemp)
status=$(curl -k -u "${roxUser}:${roxPassword}" \
  -o "$tmpOutput" \
  -w "%{http_code}\n" \
  https://"${roxEndpoint}"/v1/signatureintegrations )

if [ "${status}" != "200" ]; then
  cat "$tmpOutput"
  exit 1
fi
integrationJSON=$(jq -c -r '.integrations[0]' "$tmpOutput")
integrationID=$(echo "$integrationJSON" | jq -c -r '.id')
originalValue=$(echo "$integrationJSON" | jq -c -r '.cosign.publicKeys[0].publicKeyPemEnc')
modifiedValue="-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE04soAoNygRhaytCtygPcwsP+6Ein\nYoDv/BJx1T9WmtsANh2HplRR66Fbm+3OjFuah2IhFufPhDl6a85I3ymVYw==\n-----END PUBLIC KEY-----"

replace=$modifiedValue
while true;
do
  replacedIntegrationJSON=$(echo "$integrationJSON" | jq -c -r ".cosign.publicKeys[0].publicKeyPemEnc = \"${replace}\"")
  echo "Customized: ${replacedIntegrationJSON}"

  # Reset the value to either the original or the modified one so we continuously change it.
  if [ "$replace" = "$modifiedValue" ]; then
    replace=$originalValue
  else
    replace=$modifiedValue
  fi

  tmpOutput=$(mktemp)
  status=$(curl -k -u "${roxUser}:${roxPassword}" -X PUT \
    -d "${replacedIntegrationJSON}" \
    -o "$tmpOutput" \
    -w "%{http_code}\n" \
    https://"${roxEndpoint}"/v1/signatureintegrations/"${integrationID}" )

  if [ "${status}" != "200" ]; then
    cat "$tmpOutput"
    exit 1
  fi

 sleep 15m
done

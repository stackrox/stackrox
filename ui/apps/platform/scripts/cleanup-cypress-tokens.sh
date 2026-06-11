#!/usr/bin/env bash

# Revokes all API tokens whose names start with the cypress test token prefix.

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
roles_json="${DIR}/../cypress/constants/cypressTestRoles.json"

api_endpoint="${UI_BASE_URL:-https://localhost:8000}"
token_prefix=$(jq -r '.tokenNamePrefix' "$roles_json")

if [[ -z "$ROX_USERNAME" || -z "$ROX_ADMIN_PASSWORD" ]]; then
  source "${DIR}/../../../../scripts/k8s/export-basic-auth-creds.sh" "${DIR}/../../../../deploy/k8s"
fi

curl_cfg() {
  echo -n "$1 = \"${2//[\"\\]/\\&}\""
}

if [[ -z "$ROX_USERNAME" || -z "$ROX_ADMIN_PASSWORD" ]]; then
  echo >&2 "Cannot clean up tokens: missing ROX_USERNAME or ROX_ADMIN_PASSWORD"
  exit 1
fi

tokens_response=$(curl -sk --config <(curl_cfg user "${ROX_USERNAME}:${ROX_ADMIN_PASSWORD}") \
  "${api_endpoint}/v1/apitokens?revoked=false")

token_ids=$(echo "$tokens_response" | jq -r --arg prefix "$token_prefix" \
  '.tokens[] | select(.name | startswith($prefix)) | .id')

if [[ -z "$token_ids" ]]; then
  echo "No cypress test tokens to clean up"
  exit 0
fi

for id in $token_ids; do
  echo "Revoking token: $id"
  curl -sk --config <(curl_cfg user "${ROX_USERNAME}:${ROX_ADMIN_PASSWORD}") \
    "${api_endpoint}/v1/apitokens/revoke/${id}" \
    -X PATCH || echo >&2 "Warning: failed to revoke token $id"
done

echo "Token cleanup complete"

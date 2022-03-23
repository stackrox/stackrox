#!/bin/bash

# This creates required signature integrations to verify signatures during scale tests.

[[ -z "${ROX_PASSWORD}" ]] && die "Required env variable ROX_PASSWORD not set"
roxURL="${ROX_ENDPOINT:-https://localhost:8000}"
roxUser="${ROX_ADMIN_USER:-admin}"
roxPassword="${ROX_PASSWORD}"
integrationJSON="$1"
tmpOutput=$(mktemp)
status=$(curl -k -u "${roxUser}:${ROX_PASSWORD}" -X POST \
  -d "integrationJSON" \
  -o tmpOutput \
  - w "%{http_code}\n" \
  ${roxUrl})

if [ "${status}" != "200" ] || [ "${status}" != "429" ]; then
  cat $tmpOutput
  exit 1
fi

exit 0

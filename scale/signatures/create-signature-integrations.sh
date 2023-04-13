#!/usr/bin/env bash
set -euo pipefail

# Creates signature integrations required for signature verification.

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../tests/e2e/lib.sh
source "$TEST_ROOT/tests/e2e/lib.sh"

# Wait for central API to be reachable.
wait_for_api

require_environment "ROX_PASSWORD"
require_environment "API_ENDPOINT"

declare -a integrations=(
'{"id": "", "name": "Distroless", "cosign":{"publicKeys":[{"name": "Distroless public key", "publicKeyPemEnc":"-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEWZzVzkb8A+DbgDpaJId/bOmV8n7Q\nOqxYbK0Iro6GzSmOzxkn+N2AKawLyXi84WSwJQBK//psATakCgAQKkNTAA==\n-----END PUBLIC KEY-----"}]}}'
'{"id": "", "name": "Tekton", "cosign":{"publicKeys":[{"name": "Tekton public key", "publicKeyPemEnc":"-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEnLNw3RYx9xQjXbUEw8vonX3U4+tB\nkPnJq+zt386SCoG0ewIH5MB8+GjIDGArUULSDfjfM31Eae/71kavAUI0OA==\n-----END PUBLIC KEY-----"}]}}'
'{"id": "", "name": "Raesene", "cosign":{"publicKeys":[{"name": "Raesene public key", "publicKeyPemEnc":"-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEoeqsxUUhzWrx70u/dCAf1QgBFMVF\neyqWrtbAfwDdjONf9gbhfzURQFyZvcL7ET5PEq36x0OS9enJShKzAJKkEQ==\n-----END PUBLIC KEY-----"}]}}'
)

for integrationJSON in "${integrations[@]}"
do
  tmpOutput=$(mktemp)
  status=$(curl -k -u "admin:${ROX_PASSWORD}" -X POST \
    -d "${integrationJSON}" \
    -o "$tmpOutput" \
    -w "%{http_code}\n" \
    https://"${API_ENDPOINT}"/v1/signatureintegrations )

  if [ "${status}" != "200" ] && [ "${status}" != "429" ] && [ "${status}" != "409" ]; then
    cat "$tmpOutput"
    exit 1
  fi
done

exit 0

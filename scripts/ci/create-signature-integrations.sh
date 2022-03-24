#!/usr/bin/env bash
set -eu

# This creates required signature integrations to verify signatures during scale tests.

die() {
  echo >&2 "$@"
  exit 1
}

[[ -z "${ROX_PASSWORD}" ]] && die "Required env variable ROX_PASSWORD not set"
roxEndpoint="${API_ENDPOINT:-localhost:8000}"
roxUser="${ROX_ADMIN_USER:-admin}"
roxPassword="${ROX_PASSWORD}"

declare -a integrations=(
'{"id": "", "name": "Distroless", "cosign":{"publicKeys":[{"name": "Distroless public key", "publicKeyPemEnc":"-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEWZzVzkb8A+DbgDpaJId/bOmV8n7Q\nOqxYbK0Iro6GzSmOzxkn+N2AKawLyXi84WSwJQBK//psATakCgAQKkNTAA==\n-----END PUBLIC KEY-----"}]}}'
'{"id": "", "name": "Tekton", "cosign":{"publicKeys":[{"name": "Tekton public key", "publicKeyPemEnc":"-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEnLNw3RYx9xQjXbUEw8vonX3U4+tB\nkPnJq+zt386SCoG0ewIH5MB8+GjIDGArUULSDfjfM31Eae/71kavAUI0OA==\n-----END PUBLIC KEY-----"}]}}'
'{"id": "", "name": "Raesene", "cosign":{"publicKeys":[{"name": "Raesene public key", "publicKeyPemEnc":"-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEoeqsxUUhzWrx70u/dCAf1QgBFMVF\neyqWrtbAfwDdjONf9gbhfzURQFyZvcL7ET5PEq36x0OS9enJShKzAJKkEQ==\n-----END PUBLIC KEY-----"}]}}'
)

for integrationJSON in "${integrations[@]}"
do
  tmpOutput=$(mktemp)
  status=$(curl -k -u "${roxUser}:${roxPassword}" -X POST \
    -d "${integrationJSON}" \
    -o "$tmpOutput" \
    -w "%{http_code}\n" \
    https://"${roxEndpoint}"/v1/signatureintegrations )

  if [ "${status}" != "200" ] && [ "${status}" != "429" ] && [ "${status}" != "409" ]; then
    cat "$tmpOutput"
    exit 1
  fi
done

exit 0

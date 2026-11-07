#!/usr/bin/env bash
set -eou pipefail

# Create and enable a policy from a JSON body file by POSTing it to Central.
# Replaces the hardcoded enable-file-activity-policy*.sh scripts; the policy
# definition now lives in a standalone JSON file (see policies/*.json), referenced
# by nickname from a run config.
#
# Usage: enable-policy.sh <policy_json_file>
#   policy_json_file: path to a Central policy body (v1/policies POST payload)
#
# Requires ROX_API_TOKEN and ROX_ENDPOINT in the environment.

policy_file="${1:?usage: $0 <policy_json_file>}"

if [[ ! -f "$policy_file" ]]; then
  echo "Error: policy file not found: ${policy_file}"
  exit 1
fi

policy_name="$(jq -r '.name' "$policy_file")"

# Post the policy to Central
curl -k -X POST \
  -H "Authorization: Bearer ${ROX_API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d @"${policy_file}" \
  "${ROX_ENDPOINT}/v1/policies" || echo "Policy may already exist or failed to create"

echo "Enabled policy '${policy_name}' from ${policy_file}"

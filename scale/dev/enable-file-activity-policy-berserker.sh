#!/usr/bin/env bash
set -eoux pipefail

# Create and enable a runtime file-activity policy for berserker performance testing.
#
# The policy's "File Path" value is what actually drives file-activity monitoring:
# Sensor extracts File Path criteria from enabled runtime file-access policies and
# publishes them to the fact agent via the fact-config ConfigMap (see
# sensor/common/filesystem/settings_manager.go). This is the ONLY supported way to
# tell fact what to watch -- the old FACT_PATHS env var is deprecated and no longer
# set. So the path below determines both what fact monitors and what alerts on.
#
# Usage: enable-file-activity-policy-berserker.sh [file_path_glob] [policy_name] [description]
#   file_path_glob: doublestar GLOB for the File Path field (default /tmp/data/**,
#                   where berserker writes). NOTE it is a glob, not a regex --
#                   "/tmp/data/.*" only matches dot-files (see
#                   pkg/booleanpolicy/querybuilders valueToPathGlob); use "/tmp/data/**".
#   policy_name:    policy name (default "Berserker File Activity Test Policy")
#   description:    free-text description
#
# Examples:
#   # alerting mode: watch where berserker writes
#   ./enable-file-activity-policy-berserker.sh "/tmp/data/**" "Berserker File Activity Test Policy"
#   # idlepolicy mode: watch a directory with no berserker activity
#   ./enable-file-activity-policy-berserker.sh "/tmp/quiet/**" "Berserker Idle File Activity Policy"

file_path_glob="${1:-/tmp/data/**}"
policy_name="${2:-Berserker File Activity Test Policy}"
description="${3:-Runtime file-activity policy for berserker performance testing - triggers on file access under ${file_path_glob}}"

# Build the policy JSON. jq assembles it so the name/path/description are injected
# safely (avoids quoting issues if they ever contain spaces or special chars).
POLICY_JSON="$(jq -n \
  --arg name "$policy_name" \
  --arg description "$description" \
  --arg path "$file_path_glob" \
  '{
    name: $name,
    description: $description,
    rationale: "Matches file activity to measure performance impact",
    remediation: "This is a test policy for performance testing with berserker",
    categories: ["Anomalous Activity"],
    lifecycleStages: ["RUNTIME"],
    eventSource: "DEPLOYMENT_EVENT",
    severity: "LOW_SEVERITY",
    disabled: false,
    policySections: [
      {
        policyGroups: [
          {
            fieldName: "File Path",
            values: [ { value: $path } ]
          }
        ]
      }
    ],
    mitreAttackVectors: [],
    isDefault: false
  }')"

# Post the policy to Central
curl -k -X POST \
  -H "Authorization: Bearer ${ROX_API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "${POLICY_JSON}" \
  "${ROX_ENDPOINT}/v1/policies" || echo "Policy may already exist or failed to create"

echo "Enabled file activity policy '${policy_name}' watching ${file_path_glob}"

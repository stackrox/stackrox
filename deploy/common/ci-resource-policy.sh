#!/usr/bin/env bash
# Helm post-renderer and manifest filter for CI workloads.
set -euo pipefail

if [[ "${1:-}" == "--directory" ]]; then
    while IFS= read -r -d '' manifest; do
        "$0" < "$manifest" > "${manifest}.ci-tmp"
        mv "${manifest}.ci-tmp" "$manifest"
    done < <(find "$2" -type f \( -name '*.yaml' -o -name '*.yml' \) -print0)
    exit 0
fi

if [[ "${1:-}" == "--operator" ]]; then
    yq eval -N -j '.' - | jq -f "$(dirname "$0")/ci-operator-resources.jq" | jq -r '"---\n" + tojson'
    exit 0
fi

# Work on container objects only, preserving PVC requests and unrelated fields.
# Keep requests explicit: upgrades must replace previously defaulted requests too.
yq eval -N -j '.' - | jq -r '
  walk(
    if type == "object" and (.image? | type) == "string" and has("resources") then
      (.resources.requests.memory // .resources.limits.memory) as $memory |
      (.resources.requests.cpu // .resources.limits.cpu) as $cpu |
      if $memory != null then
        .resources.requests.memory = (if .name == "central" then "1Gi" else $memory end) |
        .resources.limits.memory = .resources.requests.memory
      else . end |
      if $cpu != null then .resources.requests.cpu = $cpu else . end |
      del(.resources.limits.cpu)
    else . end
  ) | "---\n" + tojson
'

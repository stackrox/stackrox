#!/usr/bin/env bash
#
# Build a GitHub Actions matrix of updater/stream pairs that are due
# for a refresh.
#
# Expands stream/ref pairs (from scanner-get-released-tags.sh) against
# per-stream updater lists (from source-config.yaml), then calls
# scanner-updater-check-freshness.sh for each pair. Only due pairs
# are emitted.
#
# Inputs (environment):
#   SOURCE_CONFIG — path to source-config.yaml
#
# Outputs (stdout):
#   JSON object with "has_due", "has_errors", "errors", and "matrix"
#   for GitHub Actions.
#
# Requires: yq, jq, and scanner-get-released-tags.sh and
# scanner-updater-check-freshness.sh in the same repo checkout.

set -euo pipefail

config="${SOURCE_CONFIG:?SOURCE_CONFIG is required}"

for cmd in yq jq; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "::error::$cmd is required but not found"
        exit 1
    fi
done

# Read storage location from config.
bucket=$(yq '.storage.bucket' "$config")
prefix=$(yq '.storage.prefix' "$config")

duration_to_seconds() {
    local dur="${1:?}"
    local total=0 remaining="$dur"
    while [[ -n "$remaining" ]]; do
        if [[ "$remaining" =~ ^([0-9]+)h(.*)$ ]]; then
            total=$(( total + BASH_REMATCH[1] * 3600 )); remaining="${BASH_REMATCH[2]}"
        elif [[ "$remaining" =~ ^([0-9]+)m(.*)$ ]]; then
            total=$(( total + BASH_REMATCH[1] * 60 )); remaining="${BASH_REMATCH[2]}"
        elif [[ "$remaining" =~ ^([0-9]+)s(.*)$ ]]; then
            total=$(( total + BASH_REMATCH[1] )); remaining="${BASH_REMATCH[2]}"
        else
            echo "::error::cannot parse duration: $remaining"; return 1
        fi
    done
    echo "$total"
}

default_interval=$(yq '.updaters.default.interval // "4h"' "$config")

# Expand stream/ref/updater triples, check freshness, emit due pairs.
./.github/workflows/scripts/scanner-get-released-tags.sh \
    | jq -r '.[] | "\(.version) \(.ref)"' \
    | while read -r stream ref; do
          yq ".bundles.\"$stream\" | .[]" "$config" \
              | while read -r updater; do
                    echo "$stream $ref $updater"
                done
      done \
    | while read -r stream ref updater; do
          object="gs://${bucket}/${prefix}/${stream}/sources/${updater}.json.zst"
          interval=$(yq ".updaters.\"${updater}\".interval // \"$default_interval\"" "$config")
          interval_secs=$(duration_to_seconds "$interval")

          # check-freshness.sh exit codes: 0=fresh, 1=stale/missing, 2=error.
          if GCS_OBJECT="$object" \
             ./.github/workflows/scripts/scanner-updater-check-freshness.sh --max-age "$interval_secs"
          then
              continue
          elif [[ $? -eq 1 ]]; then
              jq -n \
                  --arg source "$updater" \
                  --arg ref "$ref" \
                  --arg object "$object" \
                  '{source: $source, ref: $ref, object: $object}'
          else
              echo "::error::${object}: freshness check failed"
              jq -n --arg msg "${object}: freshness check failed" '{error: $msg}'
              continue
          fi
      done \
    | jq -s '{
          has_due:    (map(select(.source)) | length > 0 | tostring),
          has_errors: (map(select(.error))  | length > 0 | tostring),
          errors:     [.[] | .error // empty],
          matrix:     {include: [.[] | select(.source)]}
      }'

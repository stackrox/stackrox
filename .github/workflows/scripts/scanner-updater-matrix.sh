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
          object="gs://${bucket}/${prefix}/${stream}/sources/${updater}/data.json.zst"

          # check-freshness.sh exit codes: 0=fresh, 1=stale/missing, 2=error.
          if GCS_OBJECT="$object" \
             UPDATER_SOURCE="$updater" \
             SOURCE_CONFIG="$config" \
             ./.github/workflows/scripts/scanner-updater-check-freshness.sh
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

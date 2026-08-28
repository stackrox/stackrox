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

for cmd in yq jq; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "::error::$cmd is required but not found"
        exit 1
    fi
done

sc="$(dirname "$0")/scanner-updater-config.sh"

bucket=$("$sc" bucket)
prefix=$("$sc" prefix)

# Expand stream/ref/updater triples, check freshness, emit due pairs.
./.github/workflows/scripts/scanner-get-released-tags.sh \
    | jq -r '.[] | "\(.version) \(.ref)"' \
    | while read -r stream ref; do
          "$sc" sources "$stream" \
              | while read -r updater; do
                    echo "$stream $ref $updater"
                done
      done \
    | while read -r stream ref updater; do
          object="gs://${bucket}/${prefix}/${stream}/sources/${updater}.json.zst"
          interval_secs=$("$sc" interval-secs "$updater")

          fresh=$(GCS_OBJECT="$object" \
              ./.github/workflows/scripts/scanner-updater-check-freshness.sh --max-age "$interval_secs") || {
              echo "::error::${object}: freshness check failed"
              jq -n --arg msg "${object}: freshness check failed" '{error: $msg}'
              continue
          }
          if [[ "$fresh" == "true" ]]; then
              continue
          fi
          jq -n \
              --arg source "$updater" \
              --arg ref "$ref" \
              --arg object "$object" \
              '{source: $source, ref: $ref, object: $object}'
      done \
    | jq -s '{
          has_due:    (map(select(.source)) | length > 0 | tostring),
          has_errors: (map(select(.error))  | length > 0 | tostring),
          errors:     [.[] | .error // empty],
          matrix:     {include: [.[] | select(.source)]}
      }'

#!/usr/bin/env bash
# Periodically snapshot GitHub Actions cache entries and write to a log file.
# Requires GH_TOKEN and GITHUB_REPOSITORY in the environment.
#
# Usage:
#   ./.github/scripts/monitor-gha-cache.sh         # loop every 5 minutes
#   ./.github/scripts/monitor-gha-cache.sh --once   # single snapshot then exit

set -euo pipefail

LOG="/tmp/cache-monitor/cache-keys.log"
INTERVAL=300  # 5 minutes
API="https://api.github.com/repos/${GITHUB_REPOSITORY}/actions/caches"

mkdir -p "$(dirname "$LOG")"

snapshot() {
    local ts
    ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    {
        echo "=== Cache snapshot at ${ts} ==="

        # Fetch up to 100 cache entries sorted by size (largest first).
        local response
        response=$(curl -sf \
            -H "Authorization: Bearer ${GH_TOKEN}" \
            -H "Accept: application/vnd.github+json" \
            -H "X-GitHub-Api-Version: 2022-11-28" \
            "${API}?per_page=100&sort=size_in_bytes&direction=desc" 2>&1) || {
            echo "ERROR: cache API request failed: ${response}"
            echo ""
            return
        }

        # Summary line.
        local total_count total_bytes_mb
        total_count=$(jq -r '.total_count // 0' <<< "$response")
        total_bytes_mb=$(jq -r '[.actions_caches[].size_in_bytes] | add // 0 | . / 1048576 | floor' <<< "$response")
        echo "Total caches: ${total_count}  Total size: ${total_bytes_mb}MB"
        echo ""

        # Table header.
        printf "%-8s  %-20s  %-30s  %s\n" "SIZE_MB" "LAST_ACCESSED" "REF" "KEY"
        printf "%-8s  %-20s  %-30s  %s\n" "-------" "--------------------" "------------------------------" "---"

        # One line per cache entry.
        jq -r '.actions_caches[] |
            [
                (.size_in_bytes / 1048576 | floor | tostring),
                .last_accessed_at,
                .ref,
                .key
            ] | @tsv' <<< "$response" \
        | while IFS=$'\t' read -r size_mb accessed ref key; do
            printf "%-8s  %-20s  %-30s  %s\n" "${size_mb}" "${accessed}" "${ref}" "${key}"
        done

        echo ""
    } >> "$LOG"
}

if [[ "${1:-}" == "--once" ]]; then
    snapshot
    exit 0
fi

# Loop mode: take an initial snapshot, then repeat every INTERVAL seconds.
while true; do
    snapshot
    sleep "$INTERVAL"
done

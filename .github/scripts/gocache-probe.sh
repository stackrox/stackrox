#!/usr/bin/env bash
# Probe GOCACHE state and measure cache effectiveness.
# Usage:
#   gocache-probe.sh pre    # Run before the build, right after cache restore
#   gocache-probe.sh post   # Run after the build completes
#
# This script MUST NOT fail the job. All errors are caught and reported.

PHASE="${1:-pre}"
GOCACHE="$(go env GOCACHE 2>/dev/null || echo "")"
STATE_FILE="/tmp/gocache-probe-state"

if [[ -z "$GOCACHE" ]]; then
    echo "GOCACHE not available, skipping probe"
    exit 0
fi

count_entries() {
    find "$GOCACHE" -type f 2>/dev/null | wc -l | tr -d ' \n'
}

cache_size() {
    du -sh "$GOCACHE" 2>/dev/null | cut -f1 || echo "0"
}

case "$PHASE" in
    pre)
        entries=$(count_entries)
        size=$(cache_size)
        echo "entries_pre=${entries}" > "$STATE_FILE"
        echo "=== GOCACHE after restore ==="
        echo "  Entries: ${entries}"
        echo "  Size: ${size}"

        if [[ "${entries:-0}" -gt 0 ]]; then
            # Quick hit-rate probe: build a shared package and count cache misses.
            # go build -v prints each package it compiles (cache miss); cached packages are silent.
            total=$(go list -deps ./pkg/errox/... 2>/dev/null | wc -l | tr -d ' \n')
            total="${total:-0}"
            misses=$(go build -v ./pkg/errox/... 2>&1 | wc -l | tr -d ' \n')
            misses="${misses:-0}"
            if [[ "$total" -gt 0 ]]; then
                hits=$((total - misses))
                pct=$((hits * 100 / total))
                echo "  Probe (pkg/errox): ${hits}/${total} cache hits (${pct}%)"
            fi
        else
            echo "  Cache is empty — cold build"
        fi
        ;;

    post)
        entries=$(count_entries)
        size=$(cache_size)
        entries_pre=0
        if [[ -f "$STATE_FILE" ]]; then
            # shellcheck disable=SC1090
            source "$STATE_FILE"
        fi
        new_entries=$((entries - entries_pre))
        echo "=== GOCACHE after build ==="
        echo "  Entries: ${entries} (+${new_entries} new)"
        echo "  Size: ${size}"
        if [[ "$entries_pre" -gt 0 && "$entries" -gt 0 ]]; then
            reuse_pct=$((entries_pre * 100 / entries))
            echo "  Restored entries as % of final: ${reuse_pct}%"
        fi
        ;;

    *)
        echo "Usage: $0 {pre|post}" >&2
        exit 0
        ;;
esac

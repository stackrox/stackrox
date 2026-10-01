#!/usr/bin/env bash
#
# Wrapper script to run corpus collection and filtering
#
# Usage:
#   ./run-collection.sh --since 2026-08-01 --until 2026-09-01 --scope ui [--repo stackrox/stackrox]
#   ./run-collection.sh --since 2026-08-01 --until 2026-09-01 --scope go [--repo stackrox/stackrox]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Parse arguments
SCOPE="all"
REPO="stackrox/stackrox"
FORCE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --since)
            SINCE="$2"
            shift 2
            ;;
        --until)
            UNTIL="$2"
            shift 2
            ;;
        --scope)
            SCOPE="$2"
            shift 2
            ;;
        --repo)
            REPO="$2"
            shift 2
            ;;
        --force)
            FORCE="--force"
            shift
            ;;
        *)
            echo "Unknown option: $1" >&2
            echo "Usage: $0 --since YYYY-MM-DD --until YYYY-MM-DD [--scope ui|go|all] [--repo owner/name] [--force]" >&2
            exit 1
            ;;
    esac
done

# Validate required args
if [[ -z "${SINCE:-}" ]] || [[ -z "${UNTIL:-}" ]]; then
    echo "Error: --since and --until are required" >&2
    echo "Usage: $0 --since YYYY-MM-DD --until YYYY-MM-DD [--scope ui|go|all] [--repo owner/name] [--force]" >&2
    exit 1
fi

# Determine cache directory
CACHE_ROOT="${REVIEW_LINT_CACHE_ROOT:-$HOME/.cache/stackrox/review-lint}"
CACHE_DIR="$CACHE_ROOT/${SINCE}_to_${UNTIL}"

echo "==> Phase 1: Collecting PRs from GitHub" >&2
"$SCRIPT_DIR/collect-prs.py" \
    --since "$SINCE" \
    --until "$UNTIL" \
    --scope "$SCOPE" \
    --repo "$REPO" \
    $FORCE

echo "" >&2
echo "==> Phase 2: Filtering corpus" >&2
"$SCRIPT_DIR/filter-corpus.py" \
    "$CACHE_DIR" \
    --scope "$SCOPE" \
    --repo "$REPO"

echo "" >&2
echo "==> Collection complete!" >&2
echo "    Cache directory: $CACHE_DIR" >&2
echo "" >&2
echo "Output files:" >&2
echo "  - corpus.sqlite3       : Persistent corpus and triage state" >&2
echo "  - raw-prs.jsonl        : Full PR data compatibility export" >&2
echo "  - filtered-threads.jsonl : Substantive review threads compatibility export" >&2

#!/usr/bin/env bash
set -euo pipefail

# This script checks that all go.mod files in the repository have a corresponding
# dependabot configuration, and that there are no orphaned dependabot configurations
# for go.mod files that no longer exist.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
cd "$ROOT"

# Check prerequisites
if [[ ! -f .github/dependabot.yaml ]]; then
    echo "ERROR: .github/dependabot.yaml not found" >&2
    exit 1
fi

if ! command -v yq &> /dev/null; then
    echo "ERROR: yq command not found. Please install yq." >&2
    exit 1
fi

# Create temporary files for comparison
gomod_dirs_file=$(mktemp)
dependabot_dirs_file=$(mktemp)
trap 'rm -f "$gomod_dirs_file" "$dependabot_dirs_file"' EXIT

# Find all go.mod files and convert to directory paths
find . -name "go.mod" -type f | while read -r gomod_file; do
    dir=$(dirname "$gomod_file" | sed 's|^\./||; s|^\.$|/|; s|^|/|')
    # Normalize: remove trailing slash except for root
    echo "${dir%/}" | sed 's|^$|/|'
done | sort -u > "$gomod_dirs_file"

# Extract all gomod directories from dependabot.yaml and normalize
yq e '.updates[] | select(.package-ecosystem=="gomod") | .directory' .github/dependabot.yaml | \
    sed 's|/$||' | sed 's|^$|/|' | sort -u > "$dependabot_dirs_file"

# Use comm to find differences
# comm -23: lines only in file 1 (missing from dependabot)
# comm -13: lines only in file 2 (orphaned in dependabot)
missing_configs=$(comm -23 "$gomod_dirs_file" "$dependabot_dirs_file")
orphaned_configs=$(comm -13 "$gomod_dirs_file" "$dependabot_dirs_file")

exit_code=0

# Report missing configurations
if [[ -n "$missing_configs" ]]; then
    echo "ERROR: The following go.mod files do not have dependabot configurations:" >&2
    while IFS= read -r dir; do
        if [[ "$dir" == "/" ]]; then
            echo "  - ./go.mod (directory: /)" >&2
        else
            echo "  - .${dir}/go.mod (directory: ${dir})" >&2
        fi
    done <<< "$missing_configs"
    echo "" >&2
    echo "Please add a gomod update entry in .github/dependabot.yaml for each missing directory." >&2
    exit_code=1
fi

# Report orphaned configurations
if [[ -n "$orphaned_configs" ]]; then
    echo "ERROR: The following dependabot configurations refer to non-existent go.mod files:" >&2
    while IFS= read -r dir; do
        if [[ "$dir" == "/" ]]; then
            echo "  - directory: ${dir} (expected: ./go.mod)" >&2
        else
            echo "  - directory: ${dir} (expected: .${dir}/go.mod)" >&2
        fi
    done <<< "$orphaned_configs"
    echo "" >&2
    echo "Please remove these stale entries from .github/dependabot.yaml." >&2
    exit_code=1
fi

if [[ $exit_code -eq 0 ]]; then
    echo "✓ All go.mod files have corresponding dependabot configurations."
    echo "✓ No orphaned dependabot configurations found."
fi

exit $exit_code

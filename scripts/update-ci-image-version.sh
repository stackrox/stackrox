#!/usr/bin/env bash

# Update CI image version in files with AUTO-GENERATED markers
#
# Functionality:
# This script updates CI image tags (quay.io/stackrox-io/apollo-ci:stackrox-test-*)
# in configuration files by replacing the version with the current value from
# CI_IMAGE_VERSION file. It only modifies lines that immediately follow an
# AUTO-GENERATED comment marker.
#
# Usage:
#   scripts/update-ci-image-version.sh               # Update all marked files
#   scripts/update-ci-image-version.sh --check-only  # Check if updates needed
#
# Adding a new file:
# 1. Add the file path to the FILES array below
# 2. In the target file, add an AUTO-GENERATED comment before the line to update:
#
#      // AUTO-GENERATED: Updated by scripts/update-ci-image-version.sh - DO NOT EDIT MANUALLY
#      "image": "quay.io/stackrox-io/apollo-ci:stackrox-test-VERSION"

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Get CI version
CI_VERSION="$(cat "${ROOT}/CI_IMAGE_VERSION" | tr -d '\n\r ')"

# Check mode
CHECK_ONLY=false
[[ "${1:-}" == "--check-only" ]] && CHECK_ONLY=true

# Files to update
FILES=(
    ".devcontainer/devcontainer.json"
    "scale/signatures/deploy.yaml"
    ".openshift-ci/Dockerfile.build_root"
)

UPDATED=()
NEED_UPDATE=()

for file in "${FILES[@]}"; do
    file_path="${ROOT}/${file}"

    if [[ ! -f "$file_path" ]]; then
        echo "WARNING: $file not found" >&2
        continue
    fi

    # Check if file has AUTO-GENERATED marker
    if ! grep -q "AUTO-GENERATED.*update-ci-image-version.sh" "$file_path"; then
        echo "WARNING: $file missing AUTO-GENERATED marker" >&2
        continue
    fi

    # Use awk to only update lines that immediately follow AUTO-GENERATED markers
    updated_content="$(awk -v ci_version="$CI_VERSION" '
        /AUTO-GENERATED.*update-ci-image-version\.sh/ {
            print
            getline
            # Only replace the image tag on the line immediately after AUTO-GENERATED
            gsub(/quay\.io\/stackrox-io\/apollo-ci:stackrox-test-[^"'\'' ]*/, "quay.io/stackrox-io/apollo-ci:stackrox-test-" ci_version)
            print
            next
        }
        { print }
    ' "$file_path")"

    if [[ "$CHECK_ONLY" == "true" ]]; then
        if [[ "$(cat "$file_path")" != "$updated_content" ]]; then
            NEED_UPDATE+=("$file")
        fi
    else
        if [[ "$(cat "$file_path")" != "$updated_content" ]]; then
            echo "$updated_content" > "$file_path"
            UPDATED+=("$file")
        fi
    fi
done


# Output results
if [[ "$CHECK_ONLY" == "true" ]]; then
    if [[ ${#NEED_UPDATE[@]} -gt 0 ]]; then
        echo "Files need updating: ${NEED_UPDATE[*]}"
        exit 1
    else
        echo "All files up to date"
    fi
else
    if [[ ${#UPDATED[@]} -gt 0 ]]; then
        echo "Updated: ${UPDATED[*]}"
    else
        echo "No files needed updating"
    fi
fi

#!/bin/sh

# Outputs version variables consumed by go-tool.sh to generate zversion.go.
#
# Resolution order (like Go's own findgoversion()):
#   1. BUILD_VERSION_FILE env var pointing to a pre-computed version file
#   2. VERSION file in the repo root
#   3. Live git queries via make targets (requires .git directory)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Check for a pre-computed version file (for CI, Docker builds, or non-git envs).
version_file="${BUILD_VERSION_FILE:-${SCRIPT_DIR}/VERSION}"
if [ -f "$version_file" ]; then
    cat "$version_file"
    exit 0
fi

# Fall back to live git queries.
echo "STABLE_MAIN_VERSION $(make --quiet --no-print-directory tag)"
echo "STABLE_COLLECTOR_VERSION $(make --quiet --no-print-directory collector-tag)"
echo "STABLE_FACT_VERSION $(make --quiet --no-print-directory fact-tag)"
echo "STABLE_SCANNER_VERSION $(make --quiet --no-print-directory scanner-tag)"
echo "STABLE_GIT_SHORT_SHA $(make --quiet --no-print-directory shortcommit)"

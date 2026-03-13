#!/bin/sh

# Outputs version variables for go-tool.sh's XDef ldflags injection.
# Uses direct git/cat commands instead of 'make' to avoid ~15s of
# Makefile parsing overhead per invocation (5 make calls × ~3s each).

REPO_ROOT="$(cd "$(dirname "$0")" && pwd)"

main_version="${BUILD_TAG:-$(git describe --tags --abbrev=10 --dirty --long --exclude '*-nightly-*' 2>/dev/null || cat "${REPO_ROOT}/VERSION" 2>/dev/null || echo '0.0.0')}"
collector_version="$(cat "${REPO_ROOT}/COLLECTOR_VERSION" 2>/dev/null || echo '0.0.0')"
fact_version="$(cat "${REPO_ROOT}/FACT_VERSION" 2>/dev/null || echo '0.0.0')"
scanner_version="$(cat "${REPO_ROOT}/SCANNER_VERSION" 2>/dev/null || echo '0.0.0')"
git_short_sha="${SHORTCOMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"

echo "STABLE_MAIN_VERSION ${main_version}"
echo "STABLE_COLLECTOR_VERSION ${collector_version}"
echo "STABLE_FACT_VERSION ${fact_version}"
echo "STABLE_SCANNER_VERSION ${scanner_version}"
echo "STABLE_GIT_SHORT_SHA ${git_short_sha}"

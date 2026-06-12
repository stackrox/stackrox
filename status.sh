#!/bin/sh

# Note: This requires .git directory in the build context (e.g. builder container)
# Env var overrides allow CI to stabilize ldflags for test caching.
echo "STABLE_MAIN_VERSION $(make --quiet --no-print-directory tag)"
echo "STABLE_COLLECTOR_VERSION ${STABLE_COLLECTOR_VERSION:-$(make --quiet --no-print-directory collector-tag)}"
echo "STABLE_FACT_VERSION ${STABLE_FACT_VERSION:-$(make --quiet --no-print-directory fact-tag)}"
echo "STABLE_SCANNER_VERSION ${STABLE_SCANNER_VERSION:-$(make --quiet --no-print-directory scanner-tag)}"
echo "STABLE_SCANNER_V4_VULNERABILITY_VERSION $(cat scanner/VULNERABILITY_VERSION)"
echo "STABLE_GIT_SHORT_SHA $(make --quiet --no-print-directory shortcommit)"

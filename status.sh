#!/bin/sh

# Note: This requires .git directory in the build context (e.g. builder container)
echo "STABLE_MAIN_VERSION $(git describe --tags --abbrev=0 --exclude '*-nightly-*')"
echo "STABLE_COLLECTOR_VERSION $(make --quiet --no-print-directory collector-tag)"
echo "STABLE_FACT_VERSION $(make --quiet --no-print-directory fact-tag)"
echo "STABLE_SCANNER_VERSION $(make --quiet --no-print-directory scanner-tag)"
echo "STABLE_GIT_SHORT_SHA $(make --quiet --no-print-directory shortcommit)"

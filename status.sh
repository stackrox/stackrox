#!/bin/sh

# Note: This requires .git directory in the build context (e.g. builder container)
echo "STABLE_MAIN_VERSION $(make --quiet --no-print-directory tag)"
echo "STABLE_COLLECTOR_VERSION $(make --quiet --no-print-directory collector-tag)"
echo "STABLE_SCANNER_VERSION $(make --quiet --no-print-directory scanner-tag)"
echo "STABLE_GIT_SHORT_SHA $(make --quiet --no-print-directory shortcommit)"

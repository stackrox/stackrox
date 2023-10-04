#!/bin/sh

# TODO: this requires .git directory in the builder container
echo "STABLE_MAIN_VERSION $(make --quiet --no-print-directory tag)"
echo "STABLE_COLLECTOR_VERSION $(cat COLLECTOR_VERSION)"
echo "STABLE_SCANNER_VERSION $(cat SCANNER_VERSION)"
# TODO: this requires .git directory in the builder container
echo "STABLE_GIT_SHORT_SHA $(make --quiet --no-print-directory shortcommit)"

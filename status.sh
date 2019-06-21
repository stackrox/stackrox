#!/bin/sh

echo "STABLE_MAIN_VERSION $(make --quiet tag)"
echo "STABLE_COLLECTOR_VERSION $(cat COLLECTOR_VERSION)"
echo "STABLE_SCANNER_VERSION $(cat SCANNER_VERSION)"
echo "STABLE_GIT_SHORT_SHA $(git rev-parse --short HEAD)"

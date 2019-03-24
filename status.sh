#!/bin/bash

echo STABLE_MAIN_VERSION $(git describe --tags --abbrev=10 --dirty)
echo STABLE_COLLECTOR_VERSION $(cat COLLECTOR_VERSION)
echo STABLE_SCANNER_VERSION $(cat SCANNER_VERSION)
echo BUILD_TIMESTAMP "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

#!/usr/bin/env bash
set -eo pipefail

echo "Hello world"
cd ui
make ui-test

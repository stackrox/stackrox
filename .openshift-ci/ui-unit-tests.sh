#!/usr/bin/env bash
set -eo pipefail

echo "Hello world"
ls
cd ui
ls
make ui-test

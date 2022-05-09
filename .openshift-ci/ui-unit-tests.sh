#!/usr/bin/env bash
set -eox pipefail

echo "Hello world"
ls
echo "before cd ui"
cd ui
echo "befor ls"
ls
echo "before make"
make ui-test

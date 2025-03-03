#!/bin/bash
set -euo pipefail

source scripts/lib.sh

find operator/ -type f -name go.mod -printf "%h\n" | while read -r dir
do
    (info "Running \"go mod tidy\" in \"$dir\""; cd "$dir" && go mod tidy)
done

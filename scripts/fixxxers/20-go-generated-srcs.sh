#!/bin/bash
set -euo pipefail

source scripts/lib.sh

info "Running \"make go-generated-srcs\""
make go-generated-srcs

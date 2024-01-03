#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
bats \
    --print-output-on-failure \
    --verbose-run \
    --report-formatter junit \
    "${ROOT}/run-scanner-v4.bats"

#!/usr/bin/env bash
set -e

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd)"

exec "${ROOT}/scripts/roxie.sh" deploy \
    --config "${ROOT}/deploy/roxie-config-local-dev.yaml" \
    --pause-reconciliation \
    "$@"

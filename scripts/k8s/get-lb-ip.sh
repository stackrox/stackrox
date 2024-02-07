#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/lib.sh
source "$ROOT/tests/e2e/lib.sh"

get_ingress_endpoint "${1:-stackrox}" svc/central-loadbalancer '.status.loadBalancer.ingress[0] | .ip // .hostname' \
  > /dev/null

echo "${ingress_endpoint}"

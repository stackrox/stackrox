#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

run_powervs_tests() {
    info "Powervs QA e2e tests stub"

    kubectl get nodes -o wide || true
    kubectl get version || true
}

run_powervs_tests "$*"

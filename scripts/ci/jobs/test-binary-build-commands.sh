#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

make_test_bin() {
    info "Making test-bin"

    # roxctl and upgrader are extracted from their published container
    # images at runtime by ensure_roxctl_from_image (called from
    # dispatch.sh). check-workflow-run is built on demand when needed.
    # This script is kept as a no-op for backward compatibility with
    # ci-operator configs that still define test_binary_build_commands.
    info "No-op: binaries are provisioned at runtime"
}

make_test_bin "$*"

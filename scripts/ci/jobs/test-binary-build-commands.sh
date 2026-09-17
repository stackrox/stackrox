#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

make_test_bin() {
    info "Making test-bin (lightweight: roxctl extracted from image at runtime)"

    (cd ./tools/check-workflow-run && go install .)
}

make_test_bin "$*"

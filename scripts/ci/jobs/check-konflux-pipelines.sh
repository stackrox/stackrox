#!/usr/bin/env bash

# This script is intended to be run in CI, and tells you whether modifications to
# Konflux pipelines follow our expectations and conventions.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

check-konflux-pipelines() {
    echo "Ensure that modifications to our Konflux pipelines follow our expectations and conventions"

    "$ROOT/scripts/check-konflux-pipelines.sh"
}

check-konflux-pipelines

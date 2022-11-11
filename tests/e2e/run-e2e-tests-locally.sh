#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/lib.sh"

# TODO all the bits

# --flavor qa
"$ROOT/qa-tests-backend/scripts/run-part-1.sh"
# --flavor e2e
#"$ROOT/tests/e2e/run.sh"

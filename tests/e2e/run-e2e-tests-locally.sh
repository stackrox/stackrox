#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/lib.sh"

# TODO opts

# TODO teardown

# TODO has vault?

# TODO import all the e2e secrets from vualt

# --flavor qa
"$ROOT/qa-tests-backend/scripts/run-part-1.sh"
# --flavor e2e
#"$ROOT/tests/e2e/run.sh"

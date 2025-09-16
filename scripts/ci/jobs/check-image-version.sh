#!/usr/bin/env bash

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../scripts/lib.sh
source "$SCRIPTS_ROOT/scripts/lib.sh"

set -euo pipefail

info 'Ensure that all test/build image are in the same version'

"$SCRIPTS_ROOT/scripts/check-image-version.sh"

#!/usr/bin/env bash

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"

set -euo pipefail

source "$SCRIPTS_ROOT/scripts/ci/lib.sh"

binary_build_commands() {
    info "Running OSCI binary_build_commands"

    # TODO(RS-509) - this can be removed post rollout as it only serves to create the 'bin' clone of the target stackrox branch under test
}

binary_build_commands

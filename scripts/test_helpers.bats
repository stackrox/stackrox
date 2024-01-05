#!/usr/bin/env bats

# Allow to run the tests locally provided that bats-helpers are installed.
# If $BATS_CORE_ROOT is set, an attempt is made to load them from that directory,
# otherwise they are expected in the default location $HOME/bats-core.
# This makes sure that current behaviour is unaltered and existing workflows are not broken.
bats_helpers_root=${BATS_CORE_ROOT:-${HOME}/bats-core}
if [[ ! -f "${bats_helpers_root}/bats-support/load.bash" ]]; then
  # Location of bats-helpers in the CI image
  bats_helpers_root="/usr/lib/node_modules"
fi
load "${bats_helpers_root}/bats-support/load.bash"
load "${bats_helpers_root}/bats-assert/load.bash"


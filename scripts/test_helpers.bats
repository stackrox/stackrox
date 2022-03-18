#!/usr/bin/env bats

# Allow to run the tests locally provided that bats-helpers are installed in $HOME/bats-core
bats_helpers_root="${HOME}/bats-core"
if [[ ! -f "${bats_helpers_root}/bats-support/load.bash" ]]; then
  # Location of bats-helpers in the CI image
  bats_helpers_root="/usr/lib/node_modules"
fi
load "${bats_helpers_root}/bats-support/load.bash"
load "${bats_helpers_root}/bats-assert/load.bash"


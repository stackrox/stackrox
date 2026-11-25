#!/usr/bin/env bash

set -euo pipefail

bats_dir="${GITHUB_WORKSPACE}/.github/ci/bats"
npm ci --prefix "$bats_dir" --no-audit --no-fund
echo "$bats_dir/node_modules/.bin" >> "$GITHUB_PATH"

# tests/roxctl/bats-tests/helpers.bash, scripts/test_helpers.bats and
# scripts/ci/store-artifacts.bats look in /usr/lib/node_modules for CI libraries.
sudo mkdir -p /usr/lib/node_modules
for package in bats-support bats-assert; do
    sudo ln -sfnT "$bats_dir/node_modules/$package" "/usr/lib/node_modules/$package"
done

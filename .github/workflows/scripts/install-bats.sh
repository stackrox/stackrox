#!/usr/bin/env bash

set -euo pipefail

sudo npm install -g bats bats-support bats-assert
npm_root=$(sudo npm root -g)

# The existing Bats helpers look here for CI-installed libraries.
helpers_dir=/usr/lib/node_modules
if [[ "$npm_root" != "$helpers_dir" ]]; then
    sudo mkdir -p "$helpers_dir"
    for package in bats-support bats-assert; do
        sudo ln -sfn "$npm_root/$package" "$helpers_dir/$package"
    done
fi

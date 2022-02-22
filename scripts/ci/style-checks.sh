#!/usr/bin/env bash

set -euo pipefail

# Run style checks

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

source "$SCRIPTS_ROOT/scripts/lib.sh"

style_checks() {
    env | sort

    if is_GITHUB_ACTIONS; then
        require_environment "GITHUB_TOKEN"
        git config --global "url.https://${GITHUB_TOKEN}@github.com/.insteadOf" https://github.com/
    fi

    make style
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    style_checks "$*"
fi

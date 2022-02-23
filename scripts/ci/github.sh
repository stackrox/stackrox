#!/usr/bin/env bash

set -euo pipefail

# A library of reusable github actions related functions

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

source "$SCRIPTS_ROOT/scripts/lib.sh"

get_private_repo_access() {
    require_environment "ORG_TOKEN_FOR_GITHUB"
    git config --global "url.https://${ORG_TOKEN_FOR_GITHUB}:x-oauth-basic@github.com/.insteadOf" https://github.com/
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi

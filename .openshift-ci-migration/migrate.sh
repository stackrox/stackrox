#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
source "$ROOT/.openshift-ci-migration/lib.sh"

set -euo pipefail

shopt -s nullglob
for cred in /tmp/secret/**/[A-Z]*; do
    export "$(basename "$cred")"="$(cat "$cred")"
done

# For cci-export, override BASH_ENV from stackrox-test with something that is writable.
BASH_ENV=$(mktemp)
export BASH_ENV

# Clone the target repo
cd /go/src/github.com/stackrox
git clone https://github.com/stackrox/stackrox.git
cd stackrox

# Checkout the PR branch
head_ref=$(get_pr_details | jq -r '.head.ref')
info "Will try to checkout a matching PR branch using: $head_ref"
git checkout "$head_ref"

# make deps as a tire kick
make deps

# there is no build in OSCI yet, so hard code a known good one for migration
echo "3.69.x-310-g23ca4e5107" > CI_TAG

# Handoff to target repo dispatch
.openshift-ci/dispatch.sh "$*"

info "nothing to see here either"

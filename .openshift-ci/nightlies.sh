#!/usr/bin/env bash

# The handler for nightly CI defined in:
# https://github.com/openshift/release/tree/master/ci-operator/config/stackrox/stackrox/stackrox-stackrox-nightlies.yaml

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

main() {

    openshift_ci_import_creds

    gitbot(){
        git -c "user.name=RoxBot" -c "user.email=roxbot@stackrox.com" \
            -c "url.https://${GITHUB_TOKEN}:x-oauth-basic@github.com/.insteadOf=https://github.com/" \
            "${@}"
    }

    gitbot -C /tmp clone https://github.com/stackrox/stackrox.git

    cd /tmp/stackrox || {
        die "Cannot use working clone"
    }

    gitbot checkout nightlies || {
        die "Could not switch to the nightly branch"
    }

    gitbot rebase master || {
        die "Could not rebase"
    }

    [[ "$(gitbot log --oneline | head -1)" =~ an.empty.commit.for.prow.CI ]] || {
        gitbot commit --allow-empty -m 'an empty commit for prow CI'
    }

    # Create a fresh commit for prow to notice

    gitbot reset --hard HEAD~1 || {
        die "Could not delete the existing empty commit"
    }

    gitbot commit --allow-empty -m 'an empty commit for prow CI' || {
        die "Could not create an empty commit"
    }

    # Tag it

    local nightly_tag
    nightly_tag="$(gitbot describe --tags --abbrev=0 --exclude '*-nightly-*')-nightly-$(date '+%Y%m%d')"

    # Allow reruns
    (gitbot tag -d "$nightly_tag" && gitbot push --delete origin "$nightly_tag") || true

    gitbot tag "$nightly_tag" || {
        die "Could not create the tag: $nightly_tag"
    }

    # Push

    gitbot push origin "$nightly_tag" || {
        die "Could not push"
    }

    gitbot push --force || {
        die "Could not push"
    }
}

main "$@"

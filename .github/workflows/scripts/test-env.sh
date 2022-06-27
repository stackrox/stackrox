#!/bin/bash
#
# Sets some necessary variables up for local testing.
# JIRA_TOKEN still has to be set manually.
#
export GITHUB_STEP_SUMMARY=/dev/stdout
GITHUB_ACTOR="$(git config --get user.email)"
export GITHUB_ACTOR
export GITHUB_REPOSITORY=stackrox/stackrox
export GITHUB_SERVER_URL=https://github.com

export jira_project=ROX
export main_branch=master

export DRY_RUN=true

source "$(git rev-parse --show-toplevel)/.github/workflows/scripts/common.sh"

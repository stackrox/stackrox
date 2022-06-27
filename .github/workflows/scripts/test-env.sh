#!/bin/bash
set -eu

export GITHUB_STEP_SUMMARY=/dev/stdout
GITHUB_ACTOR="$(git config --get user.email)"
export GITHUB_ACTOR
export GITHUB_REPOSITORY=stackrox/stackrox
export GITHUB_SERVER_URL=https://github.com

export jira_project=ROX
export main_branch=master

export DRY_RUN=true

source "$(git rev-parse --show-toplevel)/.github/workflows/scripts/common.sh"

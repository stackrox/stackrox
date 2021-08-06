#!/usr/bin/env bash

# prepare-rh-release <path to rox repo> <source branch> <target branch>
#
# Prepares the rox repo at given path for an Operator release on RH infrastructure.
# As we're experiencing slowness/issues with Cachito, the given source branch will be used as a base for:
# - Removing all Go dependencies except the ones Operator relies on
# - Generating autogen sources
# These changes will be comitted to the given target branch, OVERWRITING anything in that branch
#
# Important: this script runs in dry-run mode by default. You MUST set DRY_RUN=false in the environment
# if you want it to actually push the target branch.
#

set -euo pipefail

run_path="$(cd "$(dirname "$0")" && pwd)"
source "${run_path}/common.sh"

git_user="roxbot"
git_user_email="roxbot@stackrox.com"
commit_msg="Add proto gen sources and remove non-operator deps"


repo_path="${1:-}"
[[ -n "$repo_path" ]] || die "No path to rox repo specified"
source_branch="${2:-}"
[[ -n "$source_branch" ]] || die "No source branch specified"
target_branch="${3:-}"
[[ -n "$target_branch" ]] || die "No target branch specified"


info "Entering rox repo at $repo_path"
cd "$repo_path"

info "Switching to $source_branch"
git checkout "$source_branch"

info "Generating sources and introducing changes"
"${run_path}/generate-proto-sources.sh" "${repo_path}"

info "Removing Non-Operator dependencies"
"${run_path}/remove-non-operator-deps.sh" "${repo_path}"

info "Adding changes to target branch: $target_branch"
# This overwrites the branch if it exists, which is ok,
# as the branch should only be used by automation anyways
git switch -C "$target_branch"
git add -A
git -c "user.name=$git_user" -c "user.email=$git_user_email" commit -am "$commit_msg"

if [[ "${DRY_RUN:-}" != "false" ]]; then
  echo
  echo "==================================================================================="
  echo " DRY RUN - NOT PUSHING ANYTHING                                                    "
  echo " You usually should not have to run this command locally. It is meant for CI only. "
  echo " If you DO need to push locally, invoke this script with DRY_RUN=false set.        "
  echo "==================================================================================="
  echo
  echo "Used $source_branch as source and would have pushed to origin/$target_branch"
  exit 0
fi

info "Pushing branch to origin/$target_branch"
git push --force origin "$target_branch"

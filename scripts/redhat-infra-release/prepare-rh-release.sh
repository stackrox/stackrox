#!/usr/bin/env bash

# prepare-rh-release <path to rox repo> <source tag> <operator target tag> <target branch>
#
# Prepares the rox repo at given path for an Operator release on RH infrastructure.
# As we're experiencing slowness/issues with Cachito, the given source tag will be used as a base for:
# - Removing all Go dependencies except the ones Operator relies on
# - Generating autogen sources
# These changes will be comitted to the given target branch, OVERWRITING anything in that branch
# The resulting commit will be tagged with the given operator target tag.
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
source_tag="${2:-}"
[[ -n "$source_tag" ]] || die "No source tag specified"
target_operator_tag="${3:-}"
[[ -n "$target_operator_tag" ]] || die "No target operator tag specified"
target_branch="${4:-}"
[[ -n "$target_branch" ]] || die "No target branch specified"


info "Entering rox repo at $repo_path"
cd "$repo_path"

info "Switching to $source_tag"
git checkout "$source_tag"

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

info "Tagging HEAD commit of $target_branch with $target_operator_tag"
git -c "user.name=$git_user" -c "user.email=$git_user_email" tag "${target_operator_tag}"

push_command="git push --dry-run"
push_tag_message="Pushing tag $target_operator_tag to remote"
push_branch_message="Pushing branch to origin/$target_branch"


if [[ "${DRY_RUN:-}" != "false" ]]; then
  echo
  echo "==================================================================================="
  echo " DRY RUN - NOT PUSHING ANYTHING                                                    "
  echo " You usually should not have to run this command locally. It is meant for CI only. "
  echo " If you DO need to push locally, invoke this script with DRY_RUN=false set.        "
  echo "==================================================================================="
  echo
  echo "Used $source_tag as source and would have pushed branch origin/$target_branch and tag $target_operator_tag"
  echo
  push_tag_message="DRY RUN: ${push_tag_message}"
  push_branch_message="DRY RUN: ${push_branch_message}"
else
  push_command="git push"  # not a dry run
fi

info "${push_tag_message}"
$push_command origin "${target_operator_tag}"

info "${push_branch_message}"
$push_command --force origin "$target_branch"

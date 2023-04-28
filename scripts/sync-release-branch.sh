#!/usr/bin/env bash

# sync-release-branch.sh
# This script checks that all commits associated with a GitHub milestone have been cherry-picked into the release
# branch. If this isn't the case, it will output a command to do so.
#
# Usage:
#   sync-release-branch.sh             - check which commits for _all_ release candidate milestones require
#                                         cherry-picking, and print instructions for doing so.
#   sync-release-branch.sh <milestone> - check which commits _up to_ the given milestone require cherry-picking,
#                                         and print instructions for doing so.
#
# Prerequisites:
#  $GH_TOKEN must be set to contain a GitHub API token with 'repo' scope OR the ~/.stackrox/workflow-config.json
#    file must exist and contain a "github_token" entry.
#  The current branch must be a release branch
#
# You can set VERBOSE=1 to get more verbose output, but this is not recommended.
#

set -eo pipefail

die() {
  echo >&2 "$@"
  exit 1
}

verbose() {
  if (( VERBOSE > 0 )); then
    echo "$@"
  fi
}

gh_curl() {
  rel_url="$1"
  shift
  full_url="https://api.github.com${rel_url}"
  if ! curl -s -H "Authorization: Bearer $GH_TOKEN" "${full_url}" "$@"; then
    die "Failed to curl GitHub API at ${full_url}"
  fi
}

# Check prerequisites

if [[ -z "$GH_TOKEN" ]]; then
  # Try to load the token from workflow config, if it is set up...
  roxhelp_bin="$(which roxhelp)"
  if [[ -x "$roxhelp_bin" ]]; then
    workflow_root="$(dirname "$roxhelp_bin")/.."
    if [[ -f "${workflow_root}/lib/github.sh" ]]; then
      # shellcheck source=/dev/null
      source "${workflow_root}/lib/github.sh"
      if [[ -n "$GITHUB_TOKEN" ]]; then
        GH_TOKEN="$GITHUB_TOKEN"
      fi
    fi
  fi
fi
[[ -n "$GH_TOKEN" ]] || die "Must set GH_TOKEN to an API token with 'repo' scope"

current_branch="$(git rev-parse --abbrev-ref HEAD)"
if [[ "$current_branch" =~ ^release/([[:digit:]]+(\.[[:digit:]]+)?\.[[:digit:]]+)\.x$ ]]; then
  current_release_family="${BASH_REMATCH[1]}"
else
  die "This does not look like a release branch: ${current_branch}"
fi

# Compile list of milestones

short_release_family="$(awk -F. '{print $(NF)}' <<<"$current_release_family")"
echo "Release family names: ${current_release_family}.x, ${short_release_family}.x"

echo "Searching GitHub milestones ..."
IFS=$'\n' read -d '' -r -a milestones < <(
  gh_curl '/repos/stackrox/stackrox/milestones?state=all&direction=desc' |
    jq --arg family "${current_release_family}" --arg shortFamily "${short_release_family}" \
      '.[] | select(.title | (startswith($family) or startswith($shortFamily))) | .title' -r |
    sort --version-sort
  ) || true

if [[ "${#milestones[@]}" -eq 0 ]]; then
  die "No milestones found for release family ${current_release_family}"
fi

arg_milestone="${1}"

if [[ -n "$arg_milestone" ]]; then
  printf '%s\n' "${milestones[@]}" | grep -F -x -q "$arg_milestone" || die "No such milestone '${arg_milestone}'! Known milestones: [${milestones[*]}]"
  milestones=("$arg_milestone")
else
  echo "Found milestones:"
  printf ' - %s\n' "${milestones[@]}"
fi

unclear_commits=()
cherrypick_commits=()
bad_cherrypick_commits=()
unclosed_prs=()
unmerged_closed_prs=()

# Do a git fetch such that we know all the commits...

echo 'Fetching all recent commits from GitHub ...'
git fetch
echo


# For each milestone, find all PRs attached to it. Then, for each PR that is closed, find the "merge" event and
# retrieve the associated commit hash. Then analyze whether this commit has already been cherry-picked or otherwise
# made it onto the branch.
for milestone in "${milestones[@]}"; do
  echo "Analyzing PRs/commits for milestone ${milestone} ..."
  milestone_prs="$(gh_curl '/search/issues?q=repo:stackrox/stackrox+is:pr+milestone:"'"$milestone"'"')"

  # Determine unclosed PRs for informational output.
  IFS=$'\n' read -d '' -r -a newly_unclosed_prs < <(
      jq <<<"$milestone_prs" \
        '.items[] | select(.state != "closed") | ("#" + (.number | tostring) + " by " + .user.login + ": " + .title)' -r) || true
  unclosed_prs+=("${newly_unclosed_prs[@]}")

  # Look
  IFS=$'\n' read -d '' -r -a closed_prs < <(
    jq <<<"$milestone_prs" '.items | sort_by(.closed_at) | .[] | select(.state == "closed") | .number' -r) || true
  for closed_pr in "${closed_prs[@]}"; do
    # GitHub will paginate events if there are more than 100. TODO: loop through pages to reliably detect all events.
    commit_id="$(
      gh_curl "/repos/stackrox/stackrox/issues/${closed_pr}/events?per_page=100" |
        jq '[.[] | select(.event == "merged") | .commit_id][0] // ""' -r)"
    if [[ -z "$commit_id" ]]; then
      unmerged_closed_prs+=("$closed_pr")
      continue
    fi

    verbose "Checking if commit ${commit_id} requires cherry-picking ..."
    if git merge-base --is-ancestor "${commit_id}" HEAD; then
      # Simple case: commit is on the branch directly
      verbose "  Oh, nice! This commit is already on the branch and wasn't even cherry-picked!"
    else
      # Search for a cherrypick log line.
      cherrypick_commit="$(git log --grep "^(cherry picked from commit ${commit_id})\$" --format='%H')"
      if [[ -n "$cherrypick_commit" ]]; then
        verbose "  This commit was cherry-picked as commit ${cherrypick_commit}"
      else
        # Search for a verbatim match on the first line. As this is not 100% precise (think very generic
        # commit messages, and somebody eliminating the PR# from the commit message), additionally check if
        # the diffs (with context removed) look the same.
        firstline="$(git log "${commit_id}...${commit_id}^" --format='%s')"
        firstline_search="${firstline//\[/\\[}"
        firstline_search="${firstline_search//\]/\\]}"
        firstline_match_commit="$(git log --grep "^${firstline_search}\$" --format='%H')"
        if [[ -n "$firstline_match_commit" ]]; then
          if cmp -s \
              <(git diff -U0 "${commit_id}" "${commit_id}^" | sed -e 's/@@.*@@/@@@@/g') \
              <(git diff -U0 "${firstline_match_commit}" "${firstline_match_commit}^" | sed -e 's/@@.*@@/@@@@/g'); then
            bad_cherrypick_commits+=("${commit_id} as ${firstline_match_commit}")
            verbose "  This commit was cherry-picked (BUT NOT WITH -x!) as ${firstline_match_commit}"
          else
            verbose "  Could not determine if ${commit_id} requires cherry-picking..."
            unclear_commits+=("${commit_id} possibly cherry-picked as ${firstline_match_commit} -- please review")
          fi
        else
          # If there is no match (neither for cherry-pick log nor for firstline), it requires cherry-picking.
          verbose "  Commit ${commit_id} requires cherry-picking"
          cherrypick_commits+=("$commit_id" "${firstline//'`'/''}")
        fi
      fi
    fi
  done
done

echo

if [[ "${#bad_cherrypick_commits[@]}" -gt 0 ]]; then
  echo "Ahem.. the following commits *were* cherry-picked, but not with '-x':"
  printf ' - %s\n' "${bad_cherrypick_commits[@]}"
  echo
fi

if [[ "${#cherrypick_commits[@]}" -gt 0 ]]; then
  echo
  echo "The following commits require cherry-picking:"
  printf ' - %s (%s)\n' "${cherrypick_commits[@]}"
  echo
  echo "Copy-pastable command:"
  echo
  echo   '  git cherry-pick -x \'
  printf '    %s   `# %s` \\\n' "${cherrypick_commits[@]}"
  echo   '    ;'
  echo
fi

if [[ "${#unclear_commits[@]}" -gt 0 ]]; then
  echo "NOTE: For the following commits, it's unclear whether they require cherry-picking:"
  printf ' - %s\n' "${unclear_commits[@]}"
  echo
fi

if [[ "${#cherrypick_commits[@]}" -eq 0 && "${#unclear_commits[@]}" -eq 0 ]]; then
  echo "No commits require cherry-picking at this point!"
  if [[ "${#unclosed_prs[@]}" -gt 0 ]]; then
    echo "HOWEVER, there are still the following unclosed PRs:"
    printf ' - %s\n' "${unclosed_prs[@]}"
    echo
  fi
elif [[ "${#unclosed_prs[@]}" -gt 0 ]]; then
  echo "The following PRs attached to a relevant milestone are not yet closed:"
  printf ' - %s\n' "${unclosed_prs[@]}"
  echo
fi

if [[ "${#unmerged_closed_prs[@]}" -gt 0 ]]; then
  echo "It looks like there were a couple of PRs attached to the milestone that were closed but not merged."
  echo "Please confirm, and remove the milestone label to make this message disappear."
  printf ' - https://github.com/stackrox/stackrox/pull/%s' "${unmerged_closed_prs[@]}"
  echo
fi

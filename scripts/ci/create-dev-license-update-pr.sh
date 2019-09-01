#!/usr/bin/env bash

set -eo pipefail

[[ -n "${GITHUB_TOKEN}" ]] || { echo >&2 "No GitHub token found"; exit 2; }

usage() {
  echo >&2 "Usage: $0 <label_name>"
  exit 2
}

branch_name="$1"

[[ -n "$branch_name" ]] || usage

pr_response_file="$(mktemp)"

message="Hello,
it is time to update our development license.
Once all CI checks pass on this PR, please approve and merge it. Afterwards, please delete the branch.

**NOTE**: If CI checks fail, the most likely reason is that the signing key needs to be rotated. Follow
the instructions on https://stack-rox.atlassian.net/wiki/spaces/ENGKB/pages/874053651 for this."

curl -sS --fail \
	-o "$pr_response_file" \
	-X POST \
	-H "Authorization: token ${GITHUB_TOKEN}" \
	'https://api.github.com/repos/stackrox/rox/pulls' \
	-d"{
	\"title\": \"Automatic update of dev license ($(date '+%Y-%m-%d'))\",
	\"body\": $(jq -sR <<<"$message"),
	\"head\": \"${branch_name}\",
	\"base\": \"master\"
}"

pr_number="$(jq <"$pr_response_file" -r '.number')"

curl -sS --fail \
	-X POST \
	-H "Authorization: token ${GITHUB_TOKEN}" \
	"https://api.github.com/repos/stackrox/rox/pulls/${pr_number}/requested_reviewers" \
	-d'{
	"team_reviewers": ["dev-license-update-approvers"]
}'

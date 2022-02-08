#!/usr/bin/env bash
# This script has derived from: rox-ci-image/.circleci/create_update_pr.sh

set -euo pipefail

[[ -n "${GITHUB_TOKEN}" ]] || { echo >&2 "Not found: GITHUB_TOKEN"; exit 2; }
[[ -n "${CIRCLE_USERNAME}" ]] || { echo >&2 "Not found: CIRCLE_USERNAME"; exit 2; }

usage() {
  echo >&2 "Usage: $0 <branch_name> <repo_name> [<pr_title>]"
  exit 2
}

branch_name="$1"
repo_name="$2"
pr_title="${3:-"Untitled"}"

[[ -n "$branch_name" ]] || usage
[[ -n "$repo_name" ]] || usage
[[ -n "$pr_title" ]] || usage

pr_response_file="$(mktemp)"

message="This is automated PR created by the CI."

status_code="$(curl -sS \
  -w '%{http_code}' \
  -o "$pr_response_file" \
  -X POST \
  -H "Authorization: token ${GITHUB_TOKEN}" \
  "https://api.github.com/repos/stackrox/${repo_name}/pulls" \
  -d"{
  \"title\": \"${pr_title}\",
  \"body\": $(jq -sR <<<"$message"),
  \"head\": \"${branch_name}\",
  \"base\": \"master\"
}")"

echo "Got status code: ${status_code}"
echo "Got PR response: $(cat "${pr_response_file}")"
# 422 is returned if the PR exists already.
[[ "${status_code}" -eq 201 || "${status_code}" -eq 422 ]]

if [[ "${status_code}" -eq 201 ]]; then
  [[ -n "${CIRCLE_USERNAME}" ]] || { echo >&2 "No CIRCLE_USERNAME found"; exit 2; }

  pr_number="$(jq <"$pr_response_file" -r '.number')"
  [[ -n "${pr_number}" ]]

  curl -sS --fail \
 -X POST \
 -H "Authorization: token ${GITHUB_TOKEN}" \
 "https://api.github.com/repos/stackrox/${repo_name}/issues/${pr_number}/assignees" \
 -d"{
    \"assignees\": [\"${CIRCLE_USERNAME}\"]
  }"
fi

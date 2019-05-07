#!/usr/bin/env bash

# This script is intended to be run in CircleCI, and tells you whether any references to tickets
# claimed to be fixed by this PR are still referenced by a TODO.

[ -n "${CIRCLE_PULL_REQUEST}" ] || { echo "Not on a PR, nothing to do!"; exit 0; }

[ -n "${GITHUB_TOKEN}" ] || { echo "No GitHub token found"; exit 2; }
[ -n "${CIRCLE_PROJECT_USERNAME}" ] || { echo "CIRCLE_PROJECT_USERNAME not found" ; exit 2; }
[ -n "${CIRCLE_PROJECT_REPONAME}" ] || { echo "CIRCLE_PROJECT_REPONAME not found" ; exit 2; }

pull_request_number="${CIRCLE_PULL_REQUEST##*/}"
url="https://api.github.com/repos/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME}/pulls/${pull_request_number}"
IFS=$'\n' read -d '' -r -a tickets < <(
	curl -sS -H "Authorization: token ${GITHUB_TOKEN}" "${url}" | jq -r '.title' | grep -Eio '\brox-[[:digit:]]+\b' | sort | uniq)

if [[ "${#tickets[@]}" == 0 ]]; then
	echo "This PR does not claim to fix any tickets!"
	exit 0
fi

echo "Tickets this PR claims to fix:"
printf " - %s\n" "${tickets[@]}"

"$(dirname "$0")/../scripts/check-todos.sh" "${tickets[@]}"

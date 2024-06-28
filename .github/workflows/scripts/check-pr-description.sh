#!/bin/bash

set -euo pipefail

PR_DESCRIPTION="$1"

EOL='
'

fail="false"

none_of=('change me!')
for pattern in "${none_of[@]}"; do
  if [[ "$PR_DESCRIPTION" = *"$EOL$pattern"* ]]; then
    gh_log error "'${pattern}' found."
    fail="true"
  fi
done

# shellcheck disable=SC2016
all_of=(
  'CHANGELOG '
  'Documentation '
  'inspected CI results'
)
for task in "${all_of[@]}"; do
  if [[ "$PR_DESCRIPTION" = *"$EOL- [x] $task"* ]]; then
    gh_log debug "'${task%% }' task is checked."
  else
    gh_log error "'${task%% }' task is not checked."
    fail="true"
  fi
done

any_of=(
  'added unit tests'
  'added e2e tests'
  'added regression tests'
  'added compatibility tests'
  'modified existing tests'
  'contributed **no automated tests**'
)
for task in "${any_of[@]}"; do
  if [[ "$PR_DESCRIPTION" = *"$EOL- [ ] $task"* ]]; then
    gh_log error "'$task' task is not checked."
    fail="true"
  fi
done

found="false"
for task in "${any_of[@]}"; do
  if [[ "$PR_DESCRIPTION" = *"$EOL- [x] $task"* ]]; then
    gh_log debug "'$task' task is checked."
    found="true"
  fi
done

if [[ "$found" = "false" ]]; then
    gh_log error 'None of the automated tests tasks are checked.'
    fail="true"
fi

if [[ "$fail" = "true" ]]; then
    exit 1
fi

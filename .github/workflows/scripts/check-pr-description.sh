#!/bin/bash

set -euo pipefail

PR_DESCRIPTION=$(cat)

EOL='
'

none_of=('change me!')
for pattern in "${none_of[@]}"; do
  if [[ "$PR_DESCRIPTION" = *"$EOL$pattern"* ]]; then
    gh_log error "'${pattern}' found."
    exit 1
  fi
done

# shellcheck disable=SC2016
all_of=(
  'CHANGELOG '
  'Documentation '
  'inspected CI results'
)
for task in "${all_of[@]}"; do
  if [[ "$PR_DESCRIPTION" != *"$EOL- \[x\] $task".* ]]; then
    gh_log debug "'${task%% }' task is checked."
  else
    gh_log error "'${task%% }' task is not checked."
    exit 1
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
    gh_log error "'${task%% }' task is not checked."
    exit 1
  fi
done
for task in "${any_of[@]}"; do
  if [[ "$PR_DESCRIPTION" = *"$EOL- [x] $task"* ]]; then
    gh_log debug "at least '$task' task is checked."
    exit 0
  fi
done
gh_log error 'none of the automated tests tasks are checked.'
exit 1

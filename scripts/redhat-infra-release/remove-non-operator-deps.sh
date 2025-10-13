#!/usr/bin/env bash

set -euo pipefail

run_path="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=./common.sh
source "${run_path}/common.sh"

repo_path="${1:-}"
[[ -n "$repo_path" ]] || die "No path to rox repo specified"

cd "$repo_path"

rm -rf ui
find . -name '*_test.go' -exec rm {} \;
{
  echo operator
  echo operator
  go list -json ./operator  | jq '.Deps[]' -r | sed 's@^vendor/@@g' | grep -E '^github\.com/stackrox/rox/' | sed -E 's@^github\.com/stackrox/rox/@@g' | sort -u | tee /dev/stdout
  find . -name '*.go' -print0 | xargs -0 -n1 dirname | sort -u | sed 's@^\./@@g' | sort -u
} | sort | uniq -u | xargs -n1 -I{} sh -c 'rm "{}/"*.go'
go mod tidy

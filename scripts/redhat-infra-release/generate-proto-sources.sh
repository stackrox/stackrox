#!/usr/bin/env bash

set -euo pipefail

run_path="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=./common.sh
source "${run_path}/common.sh"

repo_path="${1:-}"
[[ -n "$repo_path" ]] || die "No path to rox repo specified"

cd "$repo_path"

# If generated sources are already checked in, we don't need to re-generate them.
gitignore_file="$repo_path/generated/.gitignore"
[[ -f "$gitignore_file" ]] || exit 0

info "Removing generated/.gitignore"
rm "$gitignore_file"

info "Generating sources"
make proto-generated-srcs

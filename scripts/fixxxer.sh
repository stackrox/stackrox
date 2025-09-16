#!/bin/bash
# This script is an entrypoint for the action .github/workflows/fixxxer.yaml
set -euo pipefail

msg_file=$(mktemp)
trap 'rm -f "${msg_file}"' EXIT

function info()
{
  echo >&2 "$(date --iso-8601=seconds)" "$@"
}

find scripts/fixxxers -name '*.sh' -type f -printf "%p\n" | sort | while read -r script; do
    info "Running $script..."
    echo "Running $script" > "$msg_file"
    echo >> "$msg_file"
    if ! ./"$script" >> "$msg_file" 2>&1; then
        info "Script $script failed:"
        cat "$msg_file" >&2
        exit 1
    fi
    if git diff --exit-code --quiet; then
        info "No changes made by $script"
        continue
    else
        info "Committing changes made by $script"
    fi
    git commit -a -F "$msg_file"
done

#!/bin/bash
# This script is an entrypoint for the action .github/workflows/fixxxer.yaml
set -euo pipefail

msg_file=$(mktemp)
trap 'rm -f "${msg_file}"' EXIT

find scripts/fixxxers -name '*.sh' -type f -printf "%p\n" | while read -r script; do
    echo "Running $script" > "$msg_file"
    echo >> "$msg_file"
    if ! ./"$script" >> "$msg_file" 2>&1; then
        echo "Script $script failed:" >&2
        cat "$msg_file" >&2
        exit 1
    fi
    if git diff --exit-code --quiet; then
        continue
    fi
    git commit -a -F "$msg_file"
done

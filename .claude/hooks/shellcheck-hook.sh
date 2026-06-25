#!/bin/bash
# Shellcheck on Edit/Write of shell scripts
file=$(jq -r '.tool_input.file_path')
[[ "$file" != *.sh ]] && exit 0
[[ ! -f "$file" ]] && exit 0
shellcheck -S warning "$file" 2>&1 | head -15
exit 0

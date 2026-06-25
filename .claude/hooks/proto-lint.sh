#!/bin/bash
# Proto style hook — runs buf format (same as CI `make proto-style`)
file=$(jq -r '.tool_input.file_path')
[[ "$file" != *.proto ]] && exit 0
[[ ! -f "$file" ]] && exit 0
cd "$CLAUDE_PROJECT_DIR" 2>/dev/null || exit 0
if command -v buf &>/dev/null; then
  buf format --diff "$file" 2>/dev/null | head -15
fi
exit 0

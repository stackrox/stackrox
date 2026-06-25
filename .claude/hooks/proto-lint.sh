#!/bin/bash
# Proto format hook — auto-fixes formatting (same as `make proto-style` uses `buf format -w`)
file=$(jq -r '.tool_input.file_path')
[[ "$file" != *.proto ]] && exit 0
[[ ! -f "$file" ]] && exit 0
cd "$CLAUDE_PROJECT_DIR" 2>/dev/null || exit 0
command -v buf &>/dev/null && buf format -w "$file" 2>/dev/null
exit 0

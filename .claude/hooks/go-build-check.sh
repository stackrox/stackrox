#!/bin/bash
# Fast Go static checks on Edit/Write — go vet on the changed package
file=$(jq -r '.tool_input.file_path')
[[ "$file" != *.go ]] && exit 0
[[ "$file" == *"/generated/"* ]] && exit 0
[[ "$file" == *"/vendor/"* ]] && exit 0
dir=$(dirname "$file")
cd "$CLAUDE_PROJECT_DIR" 2>/dev/null || exit 0
pkg="./${dir#$CLAUDE_PROJECT_DIR/}"
go vet "$pkg" 2>&1 | head -10
exit 0

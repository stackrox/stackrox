#!/bin/bash
# Go lint hook — runs golangci-lint on the changed package (same as CI `make golangci-lint`)
# Falls back to `go vet` if golangci-lint is unavailable or misconfigured
file=$(jq -r '.tool_input.file_path')
[[ "$file" != *.go ]] && exit 0
[[ "$file" == *"/generated/"* ]] && exit 0
[[ "$file" == *"/vendor/"* ]] && exit 0
dir=$(dirname "$file")
cd "$CLAUDE_PROJECT_DIR" 2>/dev/null || exit 0
pkg="./${dir#$CLAUDE_PROJECT_DIR/}"
if command -v golangci-lint &>/dev/null; then
  golangci-lint run "$pkg" 2>&1 | head -10
else
  go vet "$pkg" 2>&1 | head -10
fi
exit 0

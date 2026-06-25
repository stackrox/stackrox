#!/bin/bash
set -o pipefail
# lint.sh — Per-file lint hook for AI coding agents (Claude Code, OpenCode, etc.)
#
# Calls the repo's own Makefile targets with FILE= or PKG= to scope checks to a
# single file. Without FILE=/PKG= the targets run on everything (CI mode).
# One source of truth for tool invocation — zero flag drift.
#
# MAKEFILE TARGETS:
#   *.go    → make golangci-lint PKG=./path/to/package
#   *.proto → make proto-style FILE=path/to/file.proto
#   *.sh    → make shell-style FILE=path/to/file.sh
#   *.ts/tsx → prettier --write (ui/apps/platform/prettier.config.js)
#
# INVOCATION:
#   Claude Code: via PostToolUse hook (reads stdin JSON)
#   OpenCode:    pass file path as $1
#   Direct:      .claude/hooks/lint.sh path/to/file.go

if [[ -n "$1" ]]; then
  file="$1"
else
  file=$(jq -r '.tool_input.file_path' 2>/dev/null)
fi
[[ -z "$file" || ! -f "$file" ]] && exit 0

PROJECT_DIR="${CLAUDE_PROJECT_DIR:-$(git rev-parse --show-toplevel 2>/dev/null)}"
[[ -z "$PROJECT_DIR" ]] && exit 0
cd "$PROJECT_DIR" || exit 0

case "$file" in
  *.go)
    [[ "$file" == *"/generated/"* || "$file" == *"/vendor/"* ]] && exit 0
    go fmt "$file" 2>/dev/null
    dir=$(dirname "$file")
    pkg="./${dir#"$PROJECT_DIR"/}"
    make -s golangci-lint PKG="$pkg" 2>&1 | head -15
    ;;
  *.proto)
    make -s proto-style FILE="$file" 2>&1 | head -15
    ;;
  *.sh)
    make -s shell-style FILE="$file" 2>&1 | head -15
    ;;
  *.ts|*.tsx|*.js|*.jsx|*.css|*.scss)
    if [[ "$file" == */ui/* ]]; then
      abs=$(cd "$(dirname "$file")" && echo "$PWD/$(basename "$file")")
      cd "$PROJECT_DIR/ui/apps/platform" 2>/dev/null && npx prettier --write "$abs" 2>/dev/null
    fi
    ;;
esac

#!/bin/bash
set -o pipefail
# lint.sh — Per-file lint hook for AI coding agents (Claude Code, OpenCode, Cursor)
#
# Calls the repo's Makefile targets to lint a single file. Uses the same tools,
# flags, and configs as CI — one source of truth, zero flag drift.
#
# MAKEFILE TARGETS:
#   *.go    → make golangci-lint-nodeps PKG=...  (gofmt + golangci-lint, no slow deps)
#   *.proto → make proto-style FILE=...          (buf format --exit-code --diff -w)
#   *.sh    → make shell-style FILE=...          (shellcheck via scripts/style/shellcheck.sh)
#   *.ts/tsx → prettier --write                  (ui/apps/platform/prettier.config.js)
#
# INVOCATION:
#   Claude Code: via PostToolUse hook (reads stdin JSON)
#   Cursor:      via .cursor/hooks.json afterFileEdit (${filePath} as $1)
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
    dir=$(dirname "$file")
    pkg="./${dir#"$PROJECT_DIR"/}"
    make -s golangci-lint-nodeps PKG="$pkg" 2>&1 | head -15
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

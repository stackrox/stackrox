#!/bin/bash
set -o pipefail
# lint.sh — Per-file lint hook for AI coding agents (Claude Code, Cursor, OpenCode)
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
#   Claude Code: via PostToolUse hook (stdin JSON with .tool_input.file_path)
#   Cursor:      reads .claude/settings.json PostToolUse (same stdin JSON format)
#   OpenCode:    pass file path as $1
#   Direct:      .claude/hooks/lint.sh path/to/file.go
#
# OUTPUT:
#   Claude Code expects plain text. Cursor expects JSON with additional_context.
#   We detect Cursor via the cursor_version field in stdin and format accordingly.

input=""
if [[ -n "$1" ]]; then
  file="$1"
else
  input=$(cat)
  file=$(echo "$input" | jq -r '.tool_input.file_path // .file_path // empty' 2>/dev/null)
fi
[[ -z "$file" || ! -f "$file" ]] && exit 0

is_cursor=$(echo "$input" | jq -r '.cursor_version // empty' 2>/dev/null)

PROJECT_DIR="${CLAUDE_PROJECT_DIR:-$(git rev-parse --show-toplevel 2>/dev/null)}"
[[ -z "$PROJECT_DIR" ]] && exit 0
cd "$PROJECT_DIR" || exit 0

lint_output=""
case "$file" in
  *.go)
    [[ "$file" == *"/generated/"* || "$file" == *"/vendor/"* ]] && exit 0
    dir=$(dirname "$file")
    pkg="./${dir#"$PROJECT_DIR"/}"
    lint_output=$(make -s golangci-lint-nodeps PKG="$pkg" 2>&1 | head -15)
    ;;
  *.proto)
    lint_output=$(make -s proto-style FILE="$file" 2>&1 | head -15)
    ;;
  *.sh)
    lint_output=$(make -s shell-style FILE="$file" 2>&1 | head -15)
    ;;
  *.ts|*.tsx|*.js|*.jsx|*.css|*.scss)
    if [[ "$file" == */ui/* ]]; then
      abs=$(cd "$(dirname "$file")" && echo "$PWD/$(basename "$file")")
      cd "$PROJECT_DIR/ui/apps/platform" 2>/dev/null && npx prettier --write "$abs" 2>/dev/null
    fi
    ;;
esac

if [[ -n "$lint_output" ]]; then
  if [[ -n "$is_cursor" ]]; then
    jq -n --arg ctx "$lint_output" '{"additional_context": $ctx}'
  else
    echo "$lint_output"
  fi
fi

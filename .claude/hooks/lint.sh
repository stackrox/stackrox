#!/bin/bash
set -o pipefail
# lint.sh — Per-file lint hook for AI coding agents (Claude Code, Cursor, OpenCode)
#
# LINT FUNNEL — three layers, same tools, narrowing scope:
#
#   Layer 1: Agent hook (this file)
#     Fires on every file write. Scoped to one file or package.
#     Auto-fixes formatting (gofmt, buf format, prettier) and surfaces static
#     analysis findings (golangci-lint, shellcheck) back to the agent.
#     Latency target: <2s for formatters, best-effort with 30s timeout for
#     golangci-lint (large packages like central/sensor/service take 4–21s).
#
#   Layer 2: Pre-commit (external — quickstyle from stackrox/workflow)
#     Fires on git commit (opt-in via tools/githooks/install-hooks.sh).
#     Runs across all changed files vs main. Covers cross-package checks this
#     hook intentionally skips: roxvet, validateimports, staticcheck, goimports,
#     eslint, newline enforcement, CircleCI config validation.
#     Not invoked by this hook — batch tools don't belong in per-write hooks.
#
#   Layer 3: CI (make style)
#     Fires on PR push. Full-repo, all targets, release-tag builds.
#     The authoritative gate — nothing merges without passing this.
#
# All three layers use the same underlying tools and configs (golangci-lint reads
# .golangci.yml, buf reads buf.yaml, etc.). The Makefile is the single source of
# truth; this hook calls Make targets with FILE= or PKG= to scope them down.
#
# MAKEFILE TARGETS:
#   *.go    → make golangci-lint-nodeps PKG=...  (gofmt + golangci-lint, no slow deps)
#   *.proto → make proto-style FILE=...          (buf format --exit-code --diff -w)
#   *.sh    → make shell-style FILE=...          (shellcheck via scripts/style/shellcheck.sh)
#   *.ts/tsx → prettier --write                  (ui/apps/platform/prettier.config.js)
#
# INVOCATION:
#   Claude Code: via PostToolUse hook (stdin JSON with .tool_input.file_path)
#   Cursor:      via postToolUse hook (stdin JSON, same format as Claude Code)
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
is_cursor=$(echo "$input" | jq -r '.cursor_version // empty' 2>/dev/null)

PROJECT_DIR="${CLAUDE_PROJECT_DIR:-$(git rev-parse --show-toplevel 2>/dev/null)}"
[[ -z "$PROJECT_DIR" ]] && exit 0
cd "$PROJECT_DIR" || exit 0

[[ -z "$file" || ! -f "$file" ]] && exit 0

lint_output=""
# Capture lint output. Suppress make/tool noise (level=, Flag --) so only findings show.
# If a tool is missing or fails to start, output is empty and we exit 0 gracefully.
# head -15 may SIGPIPE earlier pipeline stages under pipefail (exit 141);
# benign because we only use the captured stdout, never the exit code.
run_lint() { make -s "$@" 2>&1 | grep -v -e "^level=" -e "^Flag " -e "^make:" -e "^+ " | head -15; }

case "$file" in
  *.go)
    [[ "$file" == *"/generated/"* || "$file" == *"/vendor/"* ]] && exit 0
    dir=$(dirname "$file")
    pkg="./${dir#"$PROJECT_DIR"/}"
    # Large packages (e.g. central/sensor/service) can take 4–21s on first run;
    # cap at 30s so the hook doesn't block the agent indefinitely.
    lint_output=$(timeout 30 make -s golangci-lint-nodeps PKG="$pkg" 2>&1 | grep -v -e "^level=" -e "^Flag " -e "^make:" -e "^+ " | head -15)
    ;;
  *.proto)
    lint_output=$(run_lint proto-style FILE="$file")
    ;;
  *.sh)
    lint_output=$(run_lint shell-style FILE="$file")
    ;;
  *.ts|*.tsx|*.js|*.jsx|*.css|*.scss)
    if [[ "$file" == */ui/* ]]; then
      abs=$(cd "$(dirname "$file")" && echo "$PWD/$(basename "$file")")
      cd "$PROJECT_DIR/ui/apps/platform" 2>/dev/null && npx prettier --write "$abs" 2>&1 || true
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

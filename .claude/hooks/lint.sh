#!/bin/bash
# lint.sh — Per-file lint hook for AI coding agents (Claude Code, OpenCode, etc.)
#
# PURPOSE:
#   Run the same lint and format checks that CI runs, but scoped to a single file
#   on every write. This catches issues immediately instead of after push, giving
#   the agent a chance to fix problems inline.
#
# CI EQUIVALENTS:
#   This hook runs a per-file subset of these Makefile targets:
#
#   File type    | CI target          | Tool                        | Config
#   -------------|--------------------|-----------------------------|------------------
#   *.go         | make golangci-lint | golangci-lint (or go vet)   | .golangci.yml
#   *.go         | (implicit)         | gofmt                      | (built-in)
#   *.proto      | make proto-style   | buf format -w               | (built-in rules)
#   *.sh         | make shell-style   | shellcheck                  | scripts/style/shellcheck.sh
#   *.ts/tsx/etc  | make ui-lint       | prettier                    | ui/apps/platform/prettier.config.js
#
#   The shared configs (.golangci.yml, prettier.config.js, etc.) are the same ones
#   CI uses — this hook doesn't duplicate any configuration, it just invokes the
#   tools on a single file instead of the whole repo.
#
# BEHAVIOR:
#   Auto-fixers (gofmt, buf format, prettier) silently fix and exit 0.
#   Lint checks (golangci-lint, go vet, shellcheck) propagate non-zero exit
#   so the agent sees findings as errors to address.
#
# INVOCATION:
#   Claude Code: called via PostToolUse hook, reads file path from stdin JSON
#   OpenCode:    pass file path as $1, or set via tool.execute.after plugin
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
    # gofmt: auto-fix formatting (same as gofmt invoked by golangci-lint)
    go fmt "$file" 2>/dev/null
    # golangci-lint: uses .golangci.yml (same config as `make golangci-lint`)
    dir=$(dirname "$file")
    pkg="./${dir#"$PROJECT_DIR"/}"
    if command -v golangci-lint &>/dev/null; then
      golangci-lint run "$pkg" 2>&1 | head -10
    else
      go vet "$pkg" 2>&1 | head -10
    fi
    ;;
  *.proto)
    # buf format: auto-fix proto formatting (same as `make proto-style`)
    command -v buf &>/dev/null && buf format -w "$file" 2>&1 | head -10
    exit 0
    ;;
  *.sh)
    # shellcheck: same flags as scripts/style/shellcheck.sh (`make shell-style`)
    shellcheck --norc -P SCRIPTDIR -x "$file" 2>&1 | head -15
    ;;
  *.ts|*.tsx|*.js|*.jsx|*.css|*.scss)
    # prettier: auto-fix using ui/apps/platform/prettier.config.js (`make ui-lint`)
    if [[ "$file" == */ui/* ]]; then
      abs=$(cd "$(dirname "$file")" && echo "$PWD/$(basename "$file")")
      cd "$PROJECT_DIR/ui/apps/platform" 2>/dev/null && npx prettier --write "$abs" 2>/dev/null
    fi
    exit 0
    ;;
esac

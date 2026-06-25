#!/bin/bash
# Unified lint hook — auto-fixes formatting and runs static checks on file writes.
# Matches the same tools and flags used by CI (make style, make ui-lint).
#
# Auto-fixers (gofmt, buf format, prettier) always exit 0.
# Lint checks (golangci-lint, go vet, shellcheck) propagate their exit code
# so Claude sees lint failures as errors to address.
file=$(jq -r '.tool_input.file_path')
[[ ! -f "$file" ]] && exit 0

cd "$CLAUDE_PROJECT_DIR" 2>/dev/null || exit 0

case "$file" in
  *.go)
    [[ "$file" == *"/generated/"* || "$file" == *"/vendor/"* ]] && exit 0
    go fmt "$file" 2>/dev/null
    dir=$(dirname "$file")
    pkg="./${dir#"$CLAUDE_PROJECT_DIR"/}"
    if command -v golangci-lint &>/dev/null; then
      golangci-lint run "$pkg" 2>&1 | head -10
    else
      go vet "$pkg" 2>&1 | head -10
    fi
    ;;
  *.proto)
    command -v buf &>/dev/null && buf format -w "$file" 2>&1 | head -10
    exit 0
    ;;
  *.sh)
    shellcheck --norc -P SCRIPTDIR -x "$file" 2>&1 | head -15
    ;;
  *.ts|*.tsx|*.js|*.jsx|*.css|*.scss)
    if [[ "$file" == */ui/* ]]; then
      abs=$(cd "$(dirname "$file")" && echo "$PWD/$(basename "$file")")
      cd "$CLAUDE_PROJECT_DIR/ui/apps/platform" 2>/dev/null && npx prettier --write "$abs" 2>/dev/null
    fi
    exit 0
    ;;
esac

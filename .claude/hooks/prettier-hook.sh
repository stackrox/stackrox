#!/bin/bash
# Prettier auto-format hook — same formatter used by `make ui-lint` / CI
# Auto-fixes formatting on .ts/.tsx/.js/.jsx/.css/.scss files under ui/
file=$(jq -r '.tool_input.file_path')
[[ "$file" != *.ts && "$file" != *.tsx && "$file" != *.js && "$file" != *.jsx && "$file" != *.css && "$file" != *.scss ]] && exit 0
[[ "$file" != */ui/* ]] && exit 0
[[ ! -f "$file" ]] && exit 0
cd "$CLAUDE_PROJECT_DIR/ui/apps/platform" 2>/dev/null || exit 0
npx prettier --write "$file" 2>/dev/null
exit 0

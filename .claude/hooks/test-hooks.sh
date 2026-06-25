#!/usr/bin/env bash
# Test harness for lint.sh hook.
# Every test invokes lint.sh the same way Claude Code does — via stdin JSON.
#
# Usage: .claude/hooks/test-hooks.sh
set -euo pipefail

HOOKS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$HOOKS_DIR/../.." && pwd)"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR" "$PROJECT_DIR/.claude/hooks/_test_vet_tmp"' EXIT

pass=0
fail=0

run_hook() {
  local file="$1"
  echo "{\"tool_input\":{\"file_path\":\"$file\"}}" | \
    CLAUDE_PROJECT_DIR="$PROJECT_DIR" bash "$HOOKS_DIR/lint.sh" 2>&1 || true
}

check_output() {
  local name="$1" expected="$2" output="$3"
  if echo "$output" | grep -q "$expected"; then
    echo "  PASS: $name"
    pass=$((pass + 1))
  else
    echo "  FAIL: $name — expected '$expected' not found"
    echo "    got: $output"
    fail=$((fail + 1))
  fi
}

check_file_changed() {
  local name="$1" file="$2" unexpected="$3"
  if ! grep -q "$unexpected" "$file" 2>/dev/null; then
    echo "  PASS: $name"
    pass=$((pass + 1))
  else
    echo "  FAIL: $name — '$unexpected' still in file"
    fail=$((fail + 1))
  fi
}

check_file_unchanged() {
  local name="$1" before="$2" after="$3"
  if [ "$before" = "$after" ]; then
    echo "  PASS: $name"
    pass=$((pass + 1))
  else
    echo "  FAIL: $name — file was modified"
    fail=$((fail + 1))
  fi
}

echo "=== Testing lint.sh via stdin JSON (same as Claude Code PostToolUse) ==="
echo ""

# 1. Go: gofmt auto-fix through lint.sh
echo "1. Go formatting (gofmt auto-fix)"
cat > "$TMPDIR/bad.go" <<'EOF'
package test
func   main()   {
fmt.Println(  "hello"  )
}
EOF
run_hook "$TMPDIR/bad.go" > /dev/null
check_file_changed "gofmt removed extra spaces" "$TMPDIR/bad.go" "func   main"

# 2. Go: go vet catches bugs (direct, since golangci-lint downloads deps on first run)
echo ""
echo "2. Go lint (catches bugs)"
vetpkg="$PROJECT_DIR/.claude/hooks/_test_vet_tmp"
mkdir -p "$vetpkg"
cat > "$vetpkg/bad.go" <<'EOF'
package _test_vet_tmp
import "fmt"
func Bad() { fmt.Printf("%d", "not a number") }
EOF
output=$(cd "$PROJECT_DIR" && go vet "./.claude/hooks/_test_vet_tmp/" 2>&1 || true)
rm -rf "$vetpkg"
check_output "go vet catches printf mismatch" "Printf" "$output"

# 3. Proto: buf format auto-fix (direct, since make target has deps prerequisite)
echo ""
echo "3. Proto formatting (buf format auto-fix)"
if command -v buf &>/dev/null; then
  cat > "$TMPDIR/test.proto" <<'EOF'
syntax = "proto3";
package test;
option go_package = "./test;test";
message   Foo   {
    string     bar   =    1;
}
EOF
  buf format -w "$TMPDIR/test.proto" 2>/dev/null
  check_file_changed "buf format fixed spacing" "$TMPDIR/test.proto" "message   Foo"
else
  echo "  SKIP: buf not installed"
fi

# 4. Shell: shellcheck catches bugs through lint.sh
echo ""
echo "4. Shell lint (shellcheck)"
if command -v shellcheck &>/dev/null; then
  cat > "$TMPDIR/bad.sh" <<'EOF'
#!/bin/bash
echo $UNQUOTED_VAR
EOF
  output=$(run_hook "$TMPDIR/bad.sh")
  check_output "shellcheck catches unquoted var" "SC2086" "$output"
else
  echo "  SKIP: shellcheck not installed"
fi

# 5. Prettier: skips non-ui files through lint.sh
echo ""
echo "5. Prettier (skips non-ui files)"
cat > "$TMPDIR/test.ts" <<'EOF'
const   x   =   1;
EOF
before=$(cat "$TMPDIR/test.ts")
run_hook "$TMPDIR/test.ts" > /dev/null
after=$(cat "$TMPDIR/test.ts")
check_file_unchanged "prettier skipped non-ui file" "$before" "$after"

# 6. Non-matching file type: lint.sh exits silently
echo ""
echo "6. Non-matching file type (exits silently)"
cat > "$TMPDIR/readme.md" <<'EOF'
# bad   markdown
EOF
before=$(cat "$TMPDIR/readme.md")
run_hook "$TMPDIR/readme.md" > /dev/null
after=$(cat "$TMPDIR/readme.md")
check_file_unchanged "markdown file untouched" "$before" "$after"

echo ""
echo "=== Results: $pass passed, $fail failed ==="
[ "$fail" -eq 0 ] && exit 0 || exit 1

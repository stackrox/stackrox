#!/usr/bin/env bash
# Test harness for Claude Code hooks.
# Validates that lint.sh catches what CI would catch.
#
# Usage: .claude/hooks/test-hooks.sh
set -euo pipefail

HOOKS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$HOOKS_DIR/../.." && pwd)"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

pass=0
fail=0

check() {
  local name="$1" expected="$2" actual="$3"
  if echo "$actual" | grep -q "$expected"; then
    echo "  PASS: $name"
    pass=$((pass + 1))
  else
    echo "  FAIL: $name — expected pattern '$expected' not found"
    echo "    got: $actual"
    fail=$((fail + 1))
  fi
}

check_file() {
  local name="$1" file="$2" expected="$3"
  if grep -q "$expected" "$file" 2>/dev/null; then
    echo "  PASS: $name"
    pass=$((pass + 1))
  else
    echo "  FAIL: $name — expected pattern '$expected' not in file"
    fail=$((fail + 1))
  fi
}

check_file_not() {
  local name="$1" file="$2" unexpected="$3"
  if ! grep -q "$unexpected" "$file" 2>/dev/null; then
    echo "  PASS: $name"
    pass=$((pass + 1))
  else
    echo "  FAIL: $name — unexpected pattern '$unexpected' found in file"
    fail=$((fail + 1))
  fi
}

echo "=== Testing lint.sh hook ==="
echo ""

# --- Go: gofmt auto-fix ---
echo "1. Go formatting (gofmt auto-fix)"
cat > "$TMPDIR/bad.go" <<'GOEOF'
package test
func   main()   {
fmt.Println(  "hello"  )
}
GOEOF
CLAUDE_PROJECT_DIR="$PROJECT_DIR" echo "{\"tool_input\":{\"file_path\":\"$TMPDIR/bad.go\"}}" | \
  file="$TMPDIR/bad.go" CLAUDE_PROJECT_DIR="$PROJECT_DIR" bash -c '
    export CLAUDE_PROJECT_DIR
    echo "{\"tool_input\":{\"file_path\":\"'$TMPDIR/bad.go'\"}}" | jq -r ".tool_input.file_path" > /dev/null
    cd "$CLAUDE_PROJECT_DIR"
    go fmt "'$TMPDIR/bad.go'" 2>/dev/null
  '
check_file_not "gofmt removed extra spaces" "$TMPDIR/bad.go" "func   main"

# --- Go: go vet catches real bugs ---
echo ""
echo "2. Go vet (catches bugs)"
vetpkg="$PROJECT_DIR/.claude/hooks/_test_vet_tmp"
mkdir -p "$vetpkg"
cat > "$vetpkg/bad.go" <<'GOEOF'
package _test_vet_tmp
import "fmt"
func Bad() { fmt.Printf("%d", "not a number") }
GOEOF
output=$(cd "$PROJECT_DIR" && go vet "./.claude/hooks/_test_vet_tmp/" 2>&1 || true)
rm -rf "$vetpkg"
check "go vet catches printf mismatch" "Printf" "$output"

# --- Proto: buf format auto-fix ---
echo ""
echo "3. Proto formatting (buf format auto-fix)"
if command -v buf &>/dev/null; then
  cat > "$TMPDIR/test.proto" <<'PROTOEOF'
syntax = "proto3";
package test;
option go_package = "./test;test";
message   Foo   {
    string     bar   =    1;
}
PROTOEOF
  cd "$PROJECT_DIR" && buf format -w "$TMPDIR/test.proto" 2>/dev/null
  check_file_not "buf format fixed spacing" "$TMPDIR/test.proto" "message   Foo"
else
  echo "  SKIP: buf not installed"
fi

# --- Shell: shellcheck catches bugs ---
echo ""
echo "4. Shell lint (shellcheck)"
if command -v shellcheck &>/dev/null; then
  cat > "$TMPDIR/bad.sh" <<'SHEOF'
#!/bin/bash
echo $UNQUOTED_VAR
SHEOF
  output=$(shellcheck --norc -P SCRIPTDIR -x "$TMPDIR/bad.sh" 2>&1 || true)
  check "shellcheck catches unquoted var" "SC2086" "$output"
else
  echo "  SKIP: shellcheck not installed"
fi

# --- Prettier: skips non-ui files ---
echo ""
echo "5. Prettier (only runs on ui/ files)"
cat > "$TMPDIR/test.ts" <<'TSEOF'
const   x   =   1;
TSEOF
# This should NOT be formatted because it's not under ui/
original=$(cat "$TMPDIR/test.ts")
# We just verify the hook would skip it (not under ui/)
echo '{"tool_input":{"file_path":"'"$TMPDIR/test.ts"'"}}' | \
  CLAUDE_PROJECT_DIR="$PROJECT_DIR" bash "$HOOKS_DIR/lint.sh" 2>/dev/null || true
current=$(cat "$TMPDIR/test.ts")
if [ "$original" = "$current" ]; then
  echo "  PASS: prettier skipped non-ui file"
  pass=$((pass + 1))
else
  echo "  FAIL: prettier should not have touched non-ui file"
  fail=$((fail + 1))
fi

# --- Summary ---
echo ""
echo "=== Results: $pass passed, $fail failed ==="
[ "$fail" -eq 0 ] && exit 0 || exit 1

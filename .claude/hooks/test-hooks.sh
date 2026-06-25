#!/usr/bin/env bash
# Validates lint.sh works for each file type and output format.
set -euo pipefail

HOOKS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$HOOKS_DIR/../.." && pwd)"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR" "$PROJECT_DIR/.claude/hooks/_test_vet_tmp"' EXIT

pass=0 fail=0

check() {
  local name="$1" expected="$2" actual="$3"
  if echo "$actual" | grep -q "$expected"; then
    echo "  PASS: $name"; pass=$((pass + 1))
  else
    echo "  FAIL: $name — expected '$expected'"; echo "    got: $actual"; fail=$((fail + 1))
  fi
}

# --- File type checks (use tools directly — validates the tools, not the hook plumbing) ---

echo "=== File type checks ==="

echo "1. Go: gofmt"
cat > "$TMPDIR/bad.go" <<'EOF'
package test
func   main()   { }
EOF
go fmt "$TMPDIR/bad.go" >/dev/null 2>&1
grep -q "func main()" "$TMPDIR/bad.go" && { echo "  PASS: gofmt fixed"; pass=$((pass + 1)); } || { echo "  FAIL: gofmt"; fail=$((fail + 1)); }

echo "2. Go: go vet"
mkdir -p "$PROJECT_DIR/.claude/hooks/_test_vet_tmp"
echo 'package _test_vet_tmp; import "fmt"; func Bad() { fmt.Printf("%d", "x") }' > "$PROJECT_DIR/.claude/hooks/_test_vet_tmp/bad.go"
output=$(cd "$PROJECT_DIR" && go vet "./.claude/hooks/_test_vet_tmp/" 2>&1 || true)
rm -rf "$PROJECT_DIR/.claude/hooks/_test_vet_tmp"
check "go vet catches printf" "Printf" "$output"

echo "3. Proto: buf format"
cat > "$TMPDIR/test.proto" <<'EOF'
syntax = "proto3";
package test;
option go_package = "./test;test";
message   Foo   {   string   bar   =   1; }
EOF
buf format -w "$TMPDIR/test.proto" 2>/dev/null
grep -q "message Foo" "$TMPDIR/test.proto" && { echo "  PASS: buf formatted"; pass=$((pass + 1)); } || { echo "  FAIL: buf format"; fail=$((fail + 1)); }

echo "4. Shell: shellcheck"
cat > "$TMPDIR/bad.sh" <<'EOF'
#!/bin/bash
echo $UNQUOTED
EOF
output=$(shellcheck --norc -P SCRIPTDIR -x "$TMPDIR/bad.sh" 2>&1 || true)
check "shellcheck catches SC2086" "SC2086" "$output"

# --- Output format checks (validates lint.sh stdin parsing and output format) ---

echo ""
echo "=== Output format checks ==="

echo "5. Cursor: JSON output"
output=$(echo '{"tool_input":{"file_path":"'"$TMPDIR/bad.sh"'"}, "cursor_version":"3.8"}' | \
  CLAUDE_PROJECT_DIR="$PROJECT_DIR" bash "$HOOKS_DIR/lint.sh" 2>&1 || true)
check "cursor gets additional_context" "additional_context" "$output"

echo "6. Claude Code: plain text"
output=$(echo '{"tool_input":{"file_path":"'"$TMPDIR/bad.sh"'"}}' | \
  CLAUDE_PROJECT_DIR="$PROJECT_DIR" bash "$HOOKS_DIR/lint.sh" 2>&1 || true)
check "claude gets plain SC2086" "SC2086" "$output"

echo ""
echo "=== Results: $pass passed, $fail failed ==="
[ "$fail" -eq 0 ]

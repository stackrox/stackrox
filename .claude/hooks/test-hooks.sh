#!/usr/bin/env bash
# Validates lint.sh works end-to-end for each file type and output format.
# Every test calls lint.sh via stdin JSON — same path as Claude Code and Cursor.
set -euo pipefail

HOOKS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$HOOKS_DIR/../.." && pwd)"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

pass=0 fail=0

run_hook() { echo "$1" | CLAUDE_PROJECT_DIR="$PROJECT_DIR" bash "$HOOKS_DIR/lint.sh" 2>&1 || true; }
claude_json() { echo "{\"tool_input\":{\"file_path\":\"$1\"}}"; }
cursor_json() { echo "{\"tool_input\":{\"file_path\":\"$1\"}, \"cursor_version\":\"3.8\"}"; }

check() {
  local name="$1" expected="$2" actual="$3"
  if echo "$actual" | grep -q "$expected"; then
    echo "  PASS: $name"; pass=$((pass + 1))
  else
    echo "  FAIL: $name — expected '$expected'"; echo "    got: $actual"; fail=$((fail + 1))
  fi
}

# --- Shell: lint.sh catches shellcheck findings ---
echo "1. Shell file"
cat > "$TMPDIR/bad.sh" <<'EOF'
#!/bin/bash
echo $UNQUOTED
EOF
output=$(run_hook "$(claude_json "$TMPDIR/bad.sh")")
check "shellcheck via lint.sh" "SC2086" "$output"

# --- Proto: lint.sh auto-formats ---
echo "2. Proto file"
cat > "$TMPDIR/test.proto" <<'EOF'
syntax = "proto3";
package test;
option go_package = "./test;test";
message   Foo   {   string   bar   =   1; }
EOF
run_hook "$(claude_json "$TMPDIR/test.proto")" >/dev/null
grep -q "message Foo" "$TMPDIR/test.proto" && { echo "  PASS: proto format via lint.sh"; pass=$((pass + 1)); } || { echo "  FAIL: proto format"; fail=$((fail + 1)); }

# --- Non-matching: lint.sh does nothing ---
echo "3. Non-matching file"
cat > "$TMPDIR/readme.md" <<'EOF'
# test
EOF
before=$(cat "$TMPDIR/readme.md")
run_hook "$(claude_json "$TMPDIR/readme.md")" >/dev/null
after=$(cat "$TMPDIR/readme.md")
[ "$before" = "$after" ] && { echo "  PASS: markdown untouched"; pass=$((pass + 1)); } || { echo "  FAIL: markdown modified"; fail=$((fail + 1)); }

# --- Cursor output: JSON with additional_context ---
echo "4. Cursor JSON output"
output=$(run_hook "$(cursor_json "$TMPDIR/bad.sh")")
check "cursor gets additional_context" "additional_context" "$output"

# --- Claude Code output: plain text ---
echo "5. Claude Code plain text"
output=$(run_hook "$(claude_json "$TMPDIR/bad.sh")")
if echo "$output" | grep -q "additional_context"; then
  echo "  FAIL: claude should get plain text"; fail=$((fail + 1))
else
  check "claude gets plain SC2086" "SC2086" "$output"
fi

echo ""
echo "=== Results: $pass passed, $fail failed ==="
[ "$fail" -eq 0 ]

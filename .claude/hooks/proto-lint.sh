#!/bin/bash
# Proto validation hook — checks common mistakes in StackRox proto files
# Runs on every Edit/Write of .proto files. Must be fast (<200ms).
file=$(jq -r '.tool_input.file_path')
[[ "$file" != *.proto ]] && exit 0
[[ ! -f "$file" ]] && exit 0

# Required declarations
if ! grep -q '^syntax = "proto3";' "$file" 2>/dev/null; then
  echo "PROTO: missing 'syntax = \"proto3\";'"
fi
if ! grep -q '^option go_package' "$file" 2>/dev/null; then
  echo "PROTO: missing go_package option"
fi
if ! grep -q '^package ' "$file" 2>/dev/null; then
  echo "PROTO: missing package declaration"
fi

# Check for duplicate tag numbers (common copy-paste mistake)
if grep -oP '= \d+;' "$file" 2>/dev/null | sort | uniq -d | grep -q .; then
  echo "PROTO: duplicate field tag numbers detected:"
  grep -oP '= \d+;' "$file" | sort | uniq -d
fi

# Storage protos should have search tags if they're in proto/storage/
if [[ "$file" == */proto/storage/* ]] && grep -q 'message ' "$file" 2>/dev/null; then
  if ! grep -q '@gotags' "$file" 2>/dev/null && ! grep -q 'enum ' "$file" 2>/dev/null; then
    echo "PROTO: storage proto has no @gotags annotations — likely needs search/sql tags"
  fi
fi

exit 0

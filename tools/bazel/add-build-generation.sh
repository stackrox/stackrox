#!/usr/bin/env bash

# Script to add build_file_generation to a go_repository in go_deps.bzl
# Usage: ./add-build-generation.sh <repo_name>

set -euo pipefail

if [ $# -ne 1 ]; then
    echo "Usage: $0 <repo_name>"
    echo "Example: $0 com_github_googleapis_gax_go_v2"
    exit 1
fi

REPO_NAME="$1"
FILE="/Users/gualvare/sw/stackrox-2/go_deps.bzl"

echo "Adding build_file_generation to $REPO_NAME..."

# Check if it already has build_file_generation
if grep -A 10 "name = \"$REPO_NAME\"" "$FILE" | grep -q "build_file_generation"; then
    echo "✅ $REPO_NAME already has build_file_generation"
    exit 0
fi

# Add the attributes using sed
# This is a simple approach - find the go_repository block and add the attributes

python3 <<'EOF'
import sys
import re

repo_name = sys.argv[1]
file_path = sys.argv[2]

with open(file_path, 'r') as f:
    content = f.read()

# Find the go_repository block for this repo
pattern = rf'(    go_repository\(\s*name = "{repo_name}",)'
replacement = r'\1\n        build_file_generation = "on",\n        build_file_proto_mode = "disable_global",'

new_content = re.sub(pattern, replacement, content)

if new_content != content:
    with open(file_path, 'w') as f:
        f.write(new_content)
    print(f"✅ Added build_file_generation to {repo_name}")
else:
    print(f"⚠️  Could not find {repo_name} in go_deps.bzl")
    sys.exit(1)
EOF

python3 - "$REPO_NAME" "$FILE"


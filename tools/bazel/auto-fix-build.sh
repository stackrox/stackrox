#!/usr/bin/env bash

# Automatically fix common Bazel build issues by adding build_file_generation
# to packages that need it
# Usage: ./auto-fix-build.sh <target> [--config=<config>]

set -euo pipefail

TARGET="$1"
shift
CONFIG_ARGS="$@"

MAX_ITERATIONS=50
ITERATION=0

echo "======================================================================"
echo "Auto-Fix Bazel Build"
echo "======================================================================"
echo "Target: $TARGET"
echo "Config: $CONFIG_ARGS"
echo "Max iterations: $MAX_ITERATIONS"
echo ""

while [ $ITERATION -lt $MAX_ITERATIONS ]; do
    ITERATION=$((ITERATION + 1))
    echo "--- Iteration $ITERATION/$MAX_ITERATIONS ---"
    
    # Try to build and capture the error
    BUILD_OUTPUT=$(cd /Users/gualvare/sw/stackrox-2 && bazelisk build "$TARGET" $CONFIG_ARGS 2>&1 || true)
    
    # Check if build succeeded
    if echo "$BUILD_OUTPUT" | grep -q "Build completed successfully"; then
        echo ""
        echo "======================================================================"
        echo "✅ SUCCESS! Build completed on iteration $ITERATION"
        echo "======================================================================"
        exit 0
    fi
    
    # Look for "BUILD file not found" errors
    if echo "$BUILD_OUTPUT" | grep -q "BUILD file not found"; then
        # Extract the repository name from the error
        # Pattern: external repository @@repo_name//path
        # We want just repo_name
        REPO=$(echo "$BUILD_OUTPUT" | grep "external repository" | head -1 | sed -n "s/.*external repository @@\\([a-z_0-9]*\\).*/\\1/p")
        
        if [ -z "$REPO" ]; then
            echo "ERROR: Could not extract repository name from error"
            echo "$BUILD_OUTPUT" | grep "ERROR:" | head -10
            echo ""
            echo "Full error context:"
            echo "$BUILD_OUTPUT" | grep -A 2 "BUILD file not found" | head -10
            exit 1
        fi
        
        echo "Found missing BUILD file in repository: $REPO"
        echo "Adding build_file_generation..."
        
        # Add build_file_generation to the repository
        python3 - "$REPO" <<'PYTHON_SCRIPT'
import sys
import re

repo_name = sys.argv[1]
file_path = "/Users/gualvare/sw/stackrox-2/go_deps.bzl"

with open(file_path, 'r') as f:
    content = f.read()

# Find the go_repository block for this repo
# Pattern to match: go_repository(\n        name = "repo_name",
pattern = rf'(    go_repository\(\s*\n\s*name = "{repo_name}",)'
replacement = r'\1\n        build_file_generation = "on",\n        build_file_proto_mode = "disable_global",'

new_content = re.sub(pattern, replacement, content)

if new_content == content:
    print(f"⚠️  Could not find or already has build_file_generation: {repo_name}")
    sys.exit(0)  # Not fatal, continue

with open(file_path, 'w') as f:
    f.write(new_content)
print(f"✅ Added build_file_generation to {repo_name}")
PYTHON_SCRIPT
        
        # Clean the repository cache to force refetch
        echo "Cleaning $REPO cache..."
        rm -rf $(cd /Users/gualvare/sw/stackrox-2 && bazelisk info output_base 2>/dev/null)/external/$REPO
        
        echo "Retrying build..."
        continue
    fi
    
    # Check for other types of errors
    if echo "$BUILD_OUTPUT" | grep -q "ERROR:"; then
        echo "ERROR: Build failed with different error:"
        echo "$BUILD_OUTPUT" | grep "ERROR:" | head -10
        echo ""
        echo "Manual intervention may be required."
        echo "See full output in build log."
        exit 1
    fi
    
    echo "Unexpected build output. Exiting."
    exit 1
done

echo "======================================================================"
echo "❌ FAILED: Reached maximum iterations ($MAX_ITERATIONS)"
echo "======================================================================"
echo "The build still has errors after $MAX_ITERATIONS attempts."
echo "Manual review required."
exit 1


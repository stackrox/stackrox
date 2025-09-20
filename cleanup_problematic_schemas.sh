#!/bin/bash

echo "Finding and removing problematic generated schema files..."

# Get compilation errors and extract problematic files
problematic_files=$(go build ./pkg/postgres/schema 2>&1 | grep "undefined:" | awk '{print $1}' | cut -d':' -f1 | sort -u)

if [ -z "$problematic_files" ]; then
    echo "✅ No problematic files found - package compiles successfully!"
    exit 0
fi

echo "Found problematic files:"
echo "$problematic_files"

# Remove the problematic files
echo "$problematic_files" | xargs rm -f

echo "Removed problematic files. Testing compilation..."

# Test compilation
if go build ./pkg/postgres/schema 2>/dev/null; then
    echo "✅ Package now compiles successfully!"
    echo "📊 Generated schema files remaining:"
    ls pkg/postgres/schema/generated_*.go | wc -l
else
    echo "❌ Still have compilation issues. Running recursively..."
    # Recursive call to handle remaining issues
    exec "$0"
fi
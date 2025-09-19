#!/bin/bash

# Script to generate all schemas and track successes/failures
success_count=0
failure_count=0
failed_entities=()

echo "Starting batch schema generation..."

while IFS= read -r entity; do
    echo "Generating schema for: $entity"
    if ./generate-schema --entity="$entity" 2>/dev/null; then
        echo "✅ SUCCESS: $entity"
        ((success_count++))
    else
        echo "❌ FAILED: $entity"
        failed_entities+=("$entity")
        ((failure_count++))
    fi
done < all_entities.txt

echo ""
echo "============= SUMMARY ============="
echo "Total entities: $(wc -l < all_entities.txt)"
echo "Successful: $success_count"
echo "Failed: $failure_count"

if [ ${#failed_entities[@]} -gt 0 ]; then
    echo ""
    echo "Failed entities:"
    printf '%s\n' "${failed_entities[@]}"
fi

# Test compilation after generation
echo ""
echo "Testing package compilation..."
if go build ./pkg/postgres/schema 2>/dev/null; then
    echo "✅ Package compiles successfully"
else
    echo "❌ Package compilation failed"
    echo "Compilation errors:"
    go build ./pkg/postgres/schema 2>&1 | head -10
fi
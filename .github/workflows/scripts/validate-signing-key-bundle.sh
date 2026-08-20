#!/usr/bin/env bash
# Validates a Red Hat signing key bundle JSON file.
#
# Checks mirror the Go ParseKeyBundle logic in pkg/signatures/key_bundle.go:
#   - valid JSON with non-empty "cosignKeys" array
#   - each key has a non-empty "name" (after trimming) without path separators
#   - no duplicate key names
#   - each "publicKey" value is a valid PEM-encoded public key
#
# USAGE: validate-signing-key-bundle.sh <bundle.json>

set -euo pipefail

bundle="${1:?Usage: validate-signing-key-bundle.sh <bundle.json>}"

if [[ ! -f "$bundle" ]]; then
    echo "ERROR: file not found: $bundle" >&2
    exit 1
fi

# 1. Valid JSON structure with non-empty cosignKeys array.
key_count=$(jq -e '.cosignKeys | length' "$bundle") || {
    echo "ERROR: invalid JSON or missing 'cosignKeys' array" >&2
    exit 1
}

if [[ "$key_count" -eq 0 ]]; then
    echo "ERROR: key bundle must contain at least one key" >&2
    exit 1
fi

echo "Found $key_count key(s), validating..."

# 2-4. Validate each key entry.
seen_names=()
for i in $(seq 0 $((key_count - 1))); do
    raw_name=$(jq -r ".cosignKeys[$i].name // empty" "$bundle")
    pem_data=$(jq -r ".cosignKeys[$i].publicKey // empty" "$bundle")

    # Trim whitespace to match Go's strings.TrimSpace before all checks.
    name="${raw_name#"${raw_name%%[![:space:]]*}"}"
    name="${name%"${name##*[![:space:]]}"}"

    if [[ -z "$name" ]]; then
        echo "ERROR: key at index $i has empty name" >&2
        exit 1
    fi

    if [[ "$name" == */* || "$name" == *\\* ]]; then
        echo "ERROR: key name '$name' contains path separators" >&2
        exit 1
    fi

    for seen in "${seen_names[@]+"${seen_names[@]}"}"; do
        if [[ "$seen" == "$name" ]]; then
            echo "ERROR: duplicate key name '$name'" >&2
            exit 1
        fi
    done
    seen_names+=("$name")

    if [[ -z "$pem_data" ]]; then
        echo "ERROR: key '$name' has empty publicKey" >&2
        exit 1
    fi

    if ! echo "$pem_data" | openssl pkey -pubin -noout 2>/dev/null; then
        echo "ERROR: key '$name' has invalid PEM public key" >&2
        exit 1
    fi

    echo "  [$i] '$name' — valid"
done

echo "Bundle validation passed: $key_count key(s) OK"

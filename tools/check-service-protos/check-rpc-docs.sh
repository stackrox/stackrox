#!/usr/bin/env bash
# Checks that every RPC method in API service protos has a documentation comment.
# Called from 'make style-slim' via check-service-protos.

set -euo pipefail

# Find all service proto files under proto/api/
service_protos=$(git ls-files 'proto/api/**/*_service.proto')

exit_code=0

for proto in $service_protos; do
    # Read lines into an array for lookback
    mapfile -t lines < "$proto"
    total=${#lines[@]}

    for ((i = 0; i < total; i++)); do
        line="${lines[$i]}"
        # Match "rpc MethodName(" at any indentation
        if [[ "$line" =~ ^[[:space:]]*rpc[[:space:]] ]]; then
            # Check that the immediately preceding non-blank line is a comment
            found_comment=false
            j=$((i - 1))
            while ((j >= 0)); do
                prev="${lines[$j]}"
                # Skip blank lines
                if [[ "$prev" =~ ^[[:space:]]*$ ]]; then
                    ((j--))
                    continue
                fi
                # Check if it's a comment line (// or /*)
                if [[ "$prev" =~ ^[[:space:]]*/[/*] ]]; then
                    found_comment=true
                fi
                break
            done

            if [[ "$found_comment" == false ]]; then
                # Extract method name for a clearer message
                method=$(echo "$line" | sed -E 's/.*rpc[[:space:]]+([A-Za-z0-9_]+).*/\1/')
                echo "ERROR: $proto: rpc $method has no documentation comment" >&2
                exit_code=1
            fi
        fi
    done
done

if [[ "$exit_code" -ne 0 ]]; then
    echo "" >&2
    echo "Every RPC in proto/api/ must have a // comment above it describing what the endpoint does." >&2
    echo "See proto/api/v1/ping_service.proto for an example." >&2
fi

exit $exit_code

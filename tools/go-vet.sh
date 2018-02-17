#!/usr/bin/env bash
# go vet variant that respects NOVET annotations

ignored=0
total=0
status=0
while read -r line; do
    if [[ "$line" =~ ^exit\ status\ ([[:digit:]]+)$ ]]; then
        status="${BASH_REMATCH[1]}"
        continue
    fi
    total=$((total + 1))
    if [[ "$line" =~ ^([^:]+):([[:digit:]]+): ]]; then
        filename="${BASH_REMATCH[1]}"
        lineno="${BASH_REMATCH[2]}"
        line_in_file=$(tail -n+"$lineno" "$filename" | head -n1)
        if [[ "$line_in_file" =~ //\ NOVET$ ]]; then
            echo >&2 "(${line} -- IGNORED due to NOVET annotation)"
            ignored=$((ignored + 1))
            continue
        fi
    fi
    echo >&2 "$line"
done < <(go vet "$@" 2>&1)

echo "Found ${total} errors, ignored ${ignored}"
if (( total == ignored )); then
    exit 0
else
    exit "$status"
fi

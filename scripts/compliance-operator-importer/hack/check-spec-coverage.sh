#!/usr/bin/env bash
# Checks that every IMP-* requirement ID defined in specs/ appears in at least
# one *_test.go file. Reports gaps and exits non-zero when any are found.
#
# USAGE: hack/check-spec-coverage.sh
# Run from the compliance-operator-importer directory.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SPECS_DIR="$ROOT/specs"
SRC_DIR="$ROOT/internal"

# Extract unique IMP-*-NNN IDs from spec files (markdown + feature).
# Handles both "IMP-FOO-001" and range notation "IMP-FOO-001..005".
extract_spec_ids() {
    local ids=()

    # Direct IDs: IMP-XXX-NNN
    while IFS= read -r id; do
        ids+=("$id")
    done < <(grep -ohrE 'IMP-[A-Z]+-[0-9]+' "$SPECS_DIR" | sort -u)

    # Range IDs: IMP-XXX-NNN..MMM → expand to individual IDs
    while IFS= read -r range_match; do
        local prefix num_start num_end
        prefix=$(echo "$range_match" | grep -oE 'IMP-[A-Z]+-')
        num_start=$(echo "$range_match" | grep -oE '[0-9]+' | head -1)
        num_end=$(echo "$range_match" | grep -oE '[0-9]+' | tail -1)
        # Strip leading zeros for arithmetic
        local start=$((10#$num_start))
        local end=$((10#$num_end))
        local width=${#num_start}
        for ((i = start; i <= end; i++)); do
            ids+=("$(printf "%s%0${width}d" "$prefix" "$i")")
        done
    done < <(grep -ohrE 'IMP-[A-Z]+-[0-9]+\.\.[0-9]+' "$SPECS_DIR" | sort -u)

    # Deduplicate and sort
    printf '%s\n' "${ids[@]}" | sort -u
}

# Extract IDs referenced in test files.
# Matches both IMP-CLI-001 (in comments) and IMP_CLI_001 (in Go identifiers).
extract_test_ids() {
    grep -ohrE 'IMP[-_][A-Z]+[-_][0-9]+' "$SRC_DIR" --include='*_test.go' \
        | sed 's/_/-/g' \
        | sort -u
}

# IDs explicitly marked as "(removed)" in specs — no test needed.
extract_removed_ids() {
    grep -E '\(removed' "$SPECS_DIR"/*.md "$SPECS_DIR"/*.feature 2>/dev/null \
        | grep -oE 'IMP-[A-Z]+-[0-9]+' \
        | sort -u
}

spec_ids=$(extract_spec_ids)
test_ids=$(extract_test_ids)
removed_ids=$(extract_removed_ids)

# IMP-ACC-* are acceptance test IDs (real-cluster tests, not unit tests).
# They are tracked separately and excluded from the gap report.
missing=()
covered=0
skipped=0
total=0

while IFS= read -r id; do
    total=$((total + 1))

    # Skip acceptance test IDs (IMP-ACC-*)
    if [[ "$id" == IMP-ACC-* ]]; then
        skipped=$((skipped + 1))
        continue
    fi

    # Skip removed IDs
    if echo "$removed_ids" | grep -qxF "$id"; then
        skipped=$((skipped + 1))
        continue
    fi

    if echo "$test_ids" | grep -qxF "$id"; then
        covered=$((covered + 1))
    else
        missing+=("$id")
    fi
done <<< "$spec_ids"

echo "Spec coverage report"
echo "===================="
echo "Total IDs in specs:     $total"
echo "Covered by tests:       $covered"
echo "Skipped (ACC/removed):  $skipped"
echo "Missing test coverage:  ${#missing[@]}"
echo ""

if [[ ${#missing[@]} -gt 0 ]]; then
    echo "GAPS (IDs with no *_test.go reference):"
    for id in "${missing[@]}"; do
        # Show which spec file defines this ID.
        file=$(grep -rlE "\b${id}\b" "$SPECS_DIR" | head -1 | xargs basename 2>/dev/null || echo "?")
        echo "  $id  ($file)"
    done
    echo ""
    echo "FAIL: ${#missing[@]} requirement(s) lack test coverage."
    exit 1
else
    echo "OK: all testable requirements are covered."
    exit 0
fi

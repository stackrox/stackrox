#!/usr/bin/env bash

# Validate an exported vulnerability source before it can replace data in GCS
# or be included in a published bundle. Print the record count on success.
set -euo pipefail

source_file=${1:?source file is required}

if [[ ! -s "$source_file" ]]; then
    echo >&2 "missing or empty source file: $source_file"
    exit 1
fi

# Count records without holding the entire decompressed source in memory.
# A valid zstd stream with zero JSON records must fail as well.
# shellcheck disable=SC2016 # $record is a jq variable.
jq_filter='reduce inputs as $record (0; if ($record | type) == "object" then . + 1 else error("non-object JSON record") end)'
if ! record_count=$(zstd -dc "$source_file" | jq -n "$jq_filter"); then
    echo >&2 "invalid or unreadable vulnerability source: $source_file"
    exit 1
fi

if (( record_count == 0 )); then
    echo >&2 "vulnerability source contains zero records: $source_file"
    exit 1
fi

printf '%s\n' "$record_count"

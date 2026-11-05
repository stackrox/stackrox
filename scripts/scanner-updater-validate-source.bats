#!/usr/bin/env bats

setup() {
    validator="${BATS_TEST_DIRNAME}/../.github/workflows/scripts/scanner-updater-validate-source.sh"
    source_file="${BATS_TEST_TMPDIR}/source.json.zst"
}

make_source() {
    printf '%s' "$1" | zstd -q -c > "$source_file"
}

@test "counts valid JSON object records" {
    make_source $'{"id":1}\n{"id":2}\n'

    run bash "$validator" "$source_file"
    [ "$status" -eq 0 ]
    [ "$output" = "2" ]
}

@test "rejects a missing source file" {
    run bash "$validator" "$source_file"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing or empty source file"* ]]
}

@test "rejects a compressed source with zero records" {
    make_source ''

    run bash "$validator" "$source_file"
    [ "$status" -ne 0 ]
    [[ "$output" == *"zero records"* ]]
}

@test "rejects a corrupt compressed source" {
    printf 'not a zstd stream\n' > "$source_file"

    run bash "$validator" "$source_file"
    [ "$status" -ne 0 ]
    [[ "$output" == *"invalid or unreadable vulnerability source"* ]]
}

@test "rejects malformed JSON" {
    make_source $'{"id":1}\n{"id":'

    run bash "$validator" "$source_file"
    [ "$status" -ne 0 ]
    [[ "$output" == *"invalid or unreadable vulnerability source"* ]]
}

@test "rejects non-object JSON records" {
    make_source $'{"id":1}\n[]\n'

    run bash "$validator" "$source_file"
    [ "$status" -ne 0 ]
    [[ "$output" == *"invalid or unreadable vulnerability source"* ]]
}

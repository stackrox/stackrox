#!/usr/bin/env bats

setup() {
  repo_root="$BATS_TEST_DIRNAME/../../.."
  work_dir="$(mktemp -d)"
  source_dir="$work_dir/source"
  output_one="$work_dir/one/vulnerabilities.zip"
  output_two="$work_dir/two/vulnerabilities.zip"
  mkdir -p "$source_dir" "$work_dir/testdata" "$work_dir/groovy" "$work_dir/one" "$work_dir/two"
  printf '%s\n' 'CVE-2024-0001' > "$work_dir/testdata/cves.txt"
  create_fixture
}

teardown() {
  rm -rf "$work_dir"
}

create_record() {
  local kind=$1 name=$2 package=$3 did=$4 version=$5 fixed=$6 score=${7:-0}
  if [[ "$kind" == vulnerability ]]; then
    jq -cn --arg name "$name" --arg package "$package" --arg did "$did" \
      --arg version "$version" --arg fixed "$fixed" \
      '{Updater:"fixture",Fingerprint:"fixture",Date:"2026-01-01T00:00:00Z",Ref:"fixture",Kind:"vulnerability",Vuln:{name:$name,links:"https://example.test/\($name)",package:{name:$package},distribution:{did:$did,version_id:$version},fixed_in_version:$fixed}}'
  else
    jq -cn --arg name "$name" --argjson score "$score" \
      '{Updater:"fixture",Fingerprint:"fixture",Date:"2026-01-01T00:00:00Z",Ref:"fixture",Kind:"enrichment",Enrichment:{Tags:[$name],Enrichment:{id:$name,descriptions:[{value:"fixture description"}],references:[{url:"https://example.test/"}],metrics:{cvssMetricV3:[{cvssData:{baseScore:$score}}]}}}}'
  fi
}

create_fixture() {
  local alpine="$work_dir/alpine.json" debian="$work_dir/debian.json" nvd="$work_dir/nvd.json"
  {
    create_record vulnerability CVE-2024-0001 unrelated ubuntu 22.04 1
    create_record vulnerability CVE-2024-0001 busybox alpine 3.9 1
    create_record vulnerability CVE-2024-0002 busybox alpine 3.9 1
    create_record vulnerability CVE-2019-9511 nginx alpine 3.9 1
    create_record vulnerability CVE-2024-0003 nginx alpine 3.8 1
  } > "$alpine"
  create_record vulnerability CVE-2024-0001 debian-pkg debian 12 1 > "$debian"
  {
    create_record enrichment CVE-2024-0001 '' '' '' '' '' 8
    create_record enrichment CVE-2019-9511 '' '' '' '' '' 8
    create_record enrichment CVE-2017-7529 '' '' '' '' '' 4
    create_record enrichment CVE-2018-16843 '' '' '' '' '' 4
    create_record enrichment CVE-2018-16844 '' '' '' '' '' 4
    create_record enrichment CVE-2018-16845 '' '' '' '' '' 4
    create_record enrichment CVE-2019-9511 '' '' '' '' '' 8
    create_record enrichment CVE-2019-9513 '' '' '' '' '' 4
    create_record enrichment CVE-2019-9516 '' '' '' '' '' 4
    create_record enrichment CVE-2019-20372 '' '' '' '' '' 4
  } > "$nvd"
  zstd -q -T1 -f "$alpine" -o "$source_dir/alpine.json.zst"
  zstd -q -T1 -f "$debian" -o "$source_dir/debian.json.zst"
  zstd -q -T1 -f "$nvd" -o "$source_dir/nvd.json.zst"
  (cd "$source_dir" && ZIPOPT='' zip -q -X source.zip ./*.json.zst)
}

run_generator() {
  local source_zip="$source_dir/source.zip"
  if [[ $# -gt 0 && "$1" == *.zip ]]; then
    source_zip=$1
    shift
  fi
  run env SOURCE_BUNDLE_ZIP="$source_zip" \
    CI_MINIMAL_OUTPUT_PATHS="$output_one:$output_two" \
    CI_MINIMAL_TEST_CVE_PATHS="$work_dir/testdata:$work_dir/groovy" \
    "$repo_root/scanner/updater/ci/generate-ci-minimal-bundle.sh" "$@"
}

@test "filters preserve boundaries and validates required coverage" {
  run_generator "$@"
  [ "$status" -eq 0 ]
  unzip -t "$output_one"
  cmp "$output_one" "$output_two"
  unzip -p "$output_one" alpine.json.zst > "$work_dir/output-alpine.zst"
  alpine_records="$(zstd -dc "$work_dir/output-alpine.zst")"
  [[ "$alpine_records" == *'"name":"CVE-2024-0001"'* ]]
  [[ "$alpine_records" == *'"name":"CVE-2024-0002"'* ]]
  [[ "$alpine_records" == *'"name":"CVE-2019-9511"'* ]]
  [[ "$alpine_records" != *'"name":"CVE-2024-0003"'* ]]
  unzip -p "$output_one" debian.json.zst | zstd -dc > "$work_dir/output-debian.jsonl"
  [ "$(wc -l < "$work_dir/output-debian.jsonl")" -eq 1 ]
  unzip -p "$output_one" nvd.json.zst | zstd -dc > "$work_dir/output-nvd.jsonl"
  [ "$(jq -s length "$work_dir/output-nvd.jsonl")" -eq 10 ]
}

@test "empty selections, malformed records, missing members, and corrupt zstd fail safely" {
  original_one="$work_dir/original-one"
  original_two="$work_dir/original-two"
  printf one > "$output_one"
  printf two > "$output_two"
  cp "$output_one" "$original_one"
  cp "$output_two" "$original_two"

  : > "$work_dir/testdata/cves.txt"
  run_generator "$@"
  [ "$status" -ne 0 ]
  cmp "$original_one" "$output_one"
  cmp "$original_two" "$output_two"

  printf '{malformed\n' > "$work_dir/alpine.json"
  zstd -q -c "$work_dir/alpine.json" > "$source_dir/alpine.json.zst"
  (cd "$source_dir" && ZIPOPT='' zip -q -X -f source.zip alpine.json.zst)
  run_generator "$@"
  [ "$status" -ne 0 ]
  cmp "$original_one" "$output_one"
  cmp "$original_two" "$output_two"

  cp "$source_dir/source.zip" "$work_dir/missing.zip"
  (cd "$work_dir" && zip -q -d missing.zip nvd.json.zst)
  run_generator "$work_dir/missing.zip"
  [ "$status" -ne 0 ]
  cmp "$original_one" "$output_one"
  cmp "$original_two" "$output_two"

  cp "$source_dir/source.zip" "$work_dir/corrupt.zip"
  corrupt_source="$work_dir/corrupt-source"
  mkdir "$corrupt_source"
  cp "$source_dir"/*.json.zst "$corrupt_source/"
  printf corrupt > "$corrupt_source/nvd.json.zst"
  (cd "$corrupt_source" && ZIPOPT='' zip -q -X "$work_dir/corrupt.zip" ./*.json.zst)
  run_generator "$work_dir/corrupt.zip"
  [ "$status" -ne 0 ]
  cmp "$original_one" "$output_one"
  cmp "$original_two" "$output_two"
}

@test "cache reuse and URL download honor precedence" {
  cache="$work_dir/cache.zip"
  cp "$source_dir/source.zip" "$cache"
  run env SOURCE_BUNDLE_ZIP="$cache" SOURCE_BUNDLE_URL=http://127.0.0.1:1/absent \
    CI_MINIMAL_OUTPUT_PATHS="$output_one:$output_two" \
    CI_MINIMAL_TEST_CVE_PATHS="$work_dir/testdata:$work_dir/groovy" \
    "$repo_root/scanner/updater/ci/generate-ci-minimal-bundle.sh"
  [ "$status" -eq 0 ]

  download_cache="$work_dir/downloaded.zip"
  run env SOURCE_BUNDLE_ZIP="$download_cache" SOURCE_BUNDLE_URL="file://$source_dir/source.zip" \
    CI_MINIMAL_OUTPUT_PATHS="$output_one:$output_two" \
    CI_MINIMAL_TEST_CVE_PATHS="$work_dir/testdata:$work_dir/groovy" \
    "$repo_root/scanner/updater/ci/generate-ci-minimal-bundle.sh"
  [ "$status" -eq 0 ]
  [ -s "$download_cache" ]
}

@test "publication failure rolls both destinations back" {
  printf one > "$output_one"
  printf two > "$output_two"
  original_one="$(cat "$output_one")"
  original_two="$(cat "$output_two")"
  run env SOURCE_BUNDLE_ZIP="$source_dir/source.zip" \
    CI_MINIMAL_OUTPUT_PATHS="$output_one:$output_two" \
    CI_MINIMAL_TEST_CVE_PATHS="$work_dir/testdata:$work_dir/groovy" \
    CI_MINIMAL_PUBLISH_FAIL_AFTER=1 \
    "$repo_root/scanner/updater/ci/generate-ci-minimal-bundle.sh"
  [ "$status" -ne 0 ]
  [ "$(cat "$output_one")" = "$original_one" ]
  [ "$(cat "$output_two")" = "$original_two" ]
}

@test "repeated generation is identical and ignores ZIPOPT" {
  run env ZIPOPT=-9 SOURCE_BUNDLE_ZIP="$source_dir/source.zip" \
    CI_MINIMAL_OUTPUT_PATHS="$output_one:$output_two" \
    CI_MINIMAL_TEST_CVE_PATHS="$work_dir/testdata:$work_dir/groovy" \
    "$repo_root/scanner/updater/ci/generate-ci-minimal-bundle.sh" --check-reproducible
  [ "$status" -eq 0 ]
}

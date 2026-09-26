#!/usr/bin/env bats

setup() {
  repo_root="$BATS_TEST_DIRNAME/../../.."
  work_dir="$(mktemp -d)"
  source_dir="$work_dir/source"
  output_one="$work_dir/one/vulnerabilities.zip"
  output_two="$work_dir/two/vulnerabilities.zip"
  mkdir -p "$source_dir" "$work_dir/testdata" "$work_dir/groovy" "$work_dir/one" "$work_dir/two"
  printf '%s\n' 'CVE-2024-0001' > "$work_dir/testdata/cves.txt"
  printf '%s\n' '{"distributions":{"ubuntu/20.04":{"apt":true}},"repositories":{"maven":{"org.apache.struts:struts2-core":true}}}' > "$work_dir/packages.json"
  export CI_MINIMAL_PACKAGE_SELECTION="$work_dir/packages.json"
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
    create_record enrichment CVE-2017-5638 '' '' '' '' '' 10
    create_record enrichment CVE-2024-9999 '' '' '' '' '' 4
    create_record enrichment unused '' '' '' '' '' 4 | jq '.Enrichment = {Tags:null,Enrichment:null}'
  } > "$nvd"
  zstd -q -T1 -f "$alpine" -o "$source_dir/alpine.json.zst"
  zstd -q -T1 -f "$debian" -o "$source_dir/debian.json.zst"
  zstd -q -T1 -f "$nvd" -o "$source_dir/nvd.json.zst"
  {
    create_record vulnerability CVE-2024-0001 unrelated ubuntu 22.04 1 |
      jq '.Vuln.name += " on Ubuntu 22.04 LTS (jammy) - low"'
    create_record vulnerability CVE-2024-9999 apt ubuntu 20.04 1
    create_record vulnerability CVE-2024-9998 apt ubuntu 22.04 1
    create_record vulnerability CVE-2024-00010 unrelated ubuntu 22.04 1
  } | zstd -q -c > "$source_dir/ubuntu.json.zst"
  create_record vulnerability GHSA-example org.apache.struts:struts2-core '' '' 2.3.32 |
    jq '.Vuln |= (del(.distribution) + {repository:{name:"maven"}, Aliases:[{space:"CVE",name:"2017-5638"}]})' |
    zstd -q -c > "$source_dir/osv.json.zst"
  {
    zstd -dcq "$source_dir/osv.json.zst"
    create_record vulnerability CVE-2024-0001 '' '' '' 1
  } | zstd -qc > "$work_dir/osv-with-empty-package.json.zst"
  mv "$work_dir/osv-with-empty-package.json.zst" "$source_dir/osv.json.zst"
  create_record vulnerability CVE-2024-0001 manual-pkg '' '' 1 |
    jq '.Vuln |= (del(.distribution) + {repository:{name:"maven"}})' |
    zstd -q -c > "$source_dir/manual.json.zst"
  create_record vulnerability CVE-2024-0001 rhel-pkg '' '' 1 |
    jq '.Vuln |= (del(.distribution) + {repository:{name:"rhel",cpe:"cpe:2.3:o:redhat:enterprise_linux:9"},links:"https://example.test/RHSA-2024:0001"})' |
    zstd -q -c > "$source_dir/rhel-vex.json.zst"
  create_record enrichment RHSA-2024:0001 '' '' '' '' '' 8 |
    jq '.Enrichment.Enrichment |= {name:.id,severity:"Important"}' |
    zstd -q -c > "$source_dir/stackrox-rhel-csaf.json.zst"
  create_record vulnerability ALAS2-2024-2442 nss-sysinit amzn 2 1 |
    zstd -q -c > "$source_dir/aws.json.zst"
  create_record vulnerability ELSA-2024-0001 libgcrypt ol 8 1 |
    jq '.Vuln.links = "https://example.test/CVE-2024-0001.html"' |
    zstd -q -c > "$source_dir/oracle.json.zst"
  create_record vulnerability 'PHSA-2024:00001 curl Security Update.' curl photon 3.0 1 |
    jq '.Vuln.links = "https://example.test/detail?vulnId=CVE-2024-0001"' |
    zstd -q -c > "$source_dir/photon.json.zst"
  printf '%s\n' 'ALAS2-2024-2442' >> "$work_dir/testdata/cves.txt"
  (cd "$source_dir" && ZIPOPT='' zip -q -X source.zip ./*.json.zst)
}

@test "advisory identifiers and linked CVEs select native distro records" {
  run_generator
  [ "$status" -eq 0 ]
  for source in aws oracle photon; do
    unzip -p "$output_one" "$source.json.zst" | zstd -dc > "$work_dir/$source-output.jsonl"
    jq -e -s 'length > 0 and all(.[]; .Kind == "vulnerability")' "$work_dir/$source-output.jsonl"
  done
}

@test "RHEL advisory references select native CVEs and their enrichment" {
  {
    create_record vulnerability CVE-2024-7777 rhel-pkg '' '' 1 |
      jq '.Vuln.links = "https://example.test/RHSA-2024:0001"'
    create_record vulnerability CVE-2024-7777 rhel-pkg '' '' 2 |
      jq '.Vuln.Invert = true'
  } | zstd -q -c > "$source_dir/rhel-vex.json.zst"
  (cd "$source_dir" && ZIPOPT='' zip -q -X source.zip rhel-vex.json.zst)
  printf '%s\n' 'RHSA-2024:0001' >> "$work_dir/testdata/cves.txt"
  run_generator
  [ "$status" -eq 0 ]
  unzip -p "$output_one" rhel-vex.json.zst | zstd -dc > "$work_dir/rhel-output.jsonl"
  jq -e -s 'length == 2 and any(.[]; .Vuln.Invert == true)' "$work_dir/rhel-output.jsonl"
  unzip -p "$output_one" stackrox-rhel-csaf.json.zst | zstd -dc > "$work_dir/csaf-output.jsonl"
  jq -e -s 'any(.[]; .Enrichment.Enrichment.name == "RHSA-2024:0001")' "$work_dir/csaf-output.jsonl"
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
    CI_MINIMAL_PACKAGE_SELECTION="$work_dir/packages.json" \
    "$repo_root/scanner/updater/ci/generate-ci-minimal-bundle.sh" "$@"
}

@test "QA sources retain package coverage, CVE aliases, and source-specific enrichment" {
  run_generator
  [ "$status" -eq 0 ]
  unzip -p "$output_one" ubuntu.json.zst | zstd -dc > "$work_dir/ubuntu-output.jsonl"
  jq -e -s 'any(.[]; .Vuln.name == "CVE-2024-9999") and all(.[]; .Vuln.name != "CVE-2024-9998")' "$work_dir/ubuntu-output.jsonl"
  jq -e -s 'any(.[]; .Vuln.name == "CVE-2024-0001 on Ubuntu 22.04 LTS (jammy) - low") and all(.[]; .Vuln.name != "CVE-2024-00010")' "$work_dir/ubuntu-output.jsonl"
  unzip -p "$output_one" osv.json.zst | zstd -dc > "$work_dir/osv-output.jsonl"
  jq -e -s 'all(.[]; .Vuln.package.name != "")' "$work_dir/osv-output.jsonl"
  jq -e -s 'any(.[]; .Vuln.Aliases[0].name == "2017-5638" and .Vuln.repository.name == "maven" and .Vuln.distribution == null)' "$work_dir/osv-output.jsonl"
  unzip -p "$output_one" stackrox-rhel-csaf.json.zst | zstd -dc > "$work_dir/csaf-output.jsonl"
  jq -e -s 'any(.[]; .Enrichment.Enrichment.name == "RHSA-2024:0001")' "$work_dir/csaf-output.jsonl"
  unzip -p "$output_one" nvd.json.zst | zstd -dc > "$work_dir/nvd-output.jsonl"
  jq -e -s 'any(.[]; .Enrichment.Enrichment.id == "CVE-2024-9999") and any(.[]; .Enrichment.Enrichment.id == "CVE-2017-5638")' "$work_dir/nvd-output.jsonl"
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
  [ "$(jq -s length "$work_dir/output-nvd.jsonl")" -eq 12 ]
}

@test "CVE aliases select records even without a package selector" {
  printf '%s\n' 'CVE-2017-5638' >> "$work_dir/testdata/cves.txt"
  printf '%s\n' '{"distributions":{},"repositories":{}}' > "$work_dir/packages.json"
  run_generator
  [ "$status" -eq 0 ]
  unzip -p "$output_one" osv.json.zst | zstd -dc | jq -e 'select(.Vuln.name == "GHSA-example")'
}

@test "missing QA matching data or enrichment prevents publication" {
  printf '%s\n' 'CVE-2017-5638' >> "$work_dir/testdata/cves.txt"
  printf one > "$output_one"
  printf two > "$output_two"
  zstd -dcq "$source_dir/osv.json.zst" | jq 'del(.Vuln.Aliases)' | zstd -qc > "$work_dir/osv.json.zst"
  (cd "$work_dir" && ZIPOPT='' zip -q -X "$source_dir/source.zip" osv.json.zst)
  run_generator
  [ "$status" -ne 0 ]
  [[ "$output" == *'missing QA vulnerability: CVE-2017-5638'* ]]
  [ "$(cat "$output_one")" = one ]
  [ "$(cat "$output_two")" = two ]

  create_fixture
  jq 'select(.Enrichment.Enrichment.id != "CVE-2017-5638")' "$work_dir/nvd.json" | zstd -qc > "$source_dir/nvd.json.zst"
  (cd "$source_dir" && ZIPOPT='' zip -q -X source.zip nvd.json.zst)
  run_generator
  [ "$status" -ne 0 ]
  [[ "$output" == *'missing QA enrichment: CVE-2017-5638'* ]]
  [ "$(cat "$output_one")" = one ]
  [ "$(cat "$output_two")" = two ]
}

@test "production bundles prefix is supported and ambiguous members fail safely" {
  mkdir "$work_dir/prefixed"
  cp -r "$source_dir" "$work_dir/prefixed/bundles"
  (cd "$work_dir/prefixed" && ZIPOPT='' zip -q -X "$work_dir/prefixed.zip" bundles/*.json.zst)
  run_generator "$work_dir/prefixed.zip"
  [ "$status" -eq 0 ]
  cp "$output_one" "$work_dir/original.zip"
  (cd "$source_dir" && ZIPOPT='' zip -q -X "$work_dir/prefixed.zip" alpine.json.zst)
  run_generator "$work_dir/prefixed.zip"
  [ "$status" -ne 0 ]
  cmp "$output_one" "$work_dir/original.zip"
  cmp "$output_two" "$work_dir/original.zip"
}

@test "invalid envelopes among valid records and checksum mismatches fail before publication" {
  printf one > "$output_one"
  printf two > "$output_two"
  printf '%s\n' '{"Kind":"vulnerability"}' >> "$work_dir/alpine.json"
  zstd -q -c "$work_dir/alpine.json" > "$source_dir/alpine.json.zst"
  (cd "$source_dir" && ZIPOPT='' zip -q -X source.zip alpine.json.zst)
  run_generator
  [ "$status" -ne 0 ]
  [ "$(cat "$output_one")" = one ]
  [ "$(cat "$output_two")" = two ]
  export SOURCE_BUNDLE_SHA256=invalid
  run_generator
  [ "$status" -ne 0 ]
  [[ "$output" == *'source SHA256 mismatch'* ]]
  [ "$(cat "$output_one")" = one ]
  [ "$(cat "$output_two")" = two ]
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
  (cd "$source_dir" && ZIPOPT='' zip -q -X source.zip alpine.json.zst)
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

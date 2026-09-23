#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/../../.." && pwd)
cd "$repo_root"

# ---- Bundle configuration -------------------------------------------------

source_bundle_url="${SOURCE_BUNDLE_URL:-https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulnerabilities.zip}"
# Set this when the mutable development URL is pinned to a saved archive.
source_bundle_sha256="${SOURCE_BUNDLE_SHA256:-}"

source_members=(
  "alpine.json.zst"
  "debian.json.zst"
  "ubuntu.json.zst"
  "osv.json.zst"
  "manual.json.zst"
  "rhel-vex.json.zst"
  "aws.json.zst"
  "oracle.json.zst"
  "photon.json.zst"
  "stackrox-rhel-csaf.json.zst"
  "nvd.json.zst"
)
source_selections=(
  alpine
  test-cves
  packages
  packages
  all
  test-cves
  test-cves
  test-cves
  test-cves
  enrichment
  enrichment
)

package_selection="${CI_MINIMAL_PACKAGE_SELECTION:-scanner/updater/ci/qa-packages.json}"

test_cve_paths=(
  "scanner/e2etests/testdata/image_tests.json"
  "qa-tests-backend/src/test/groovy"
)

additional_cves=(
  CVE-2017-7529
  CVE-2018-16843
  CVE-2018-16844
  CVE-2018-16845
  CVE-2019-9511
  CVE-2019-9513
  CVE-2019-9516
  CVE-2019-20372
)

qa_cves=(
  CVE-2017-5638
  CVE-2025-15467
  CVE-2021-33910
  CVE-2023-4911
  CVE-2025-11468
  CVE-2022-3219
)

alpine_distribution_id="alpine"
alpine_version_id="3.9"
alpine_packages=(
  busybox
  freetype
  libgcrypt
  libjpeg-turbo
  libpng
  libxml2
  libxslt
  musl
  nginx
  pcre
)

output_paths=(
  "scanner/updater/ci/bundles/ci-minimal/vulnerabilities.zip"
  "scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip"
)

if [[ -n "${CI_MINIMAL_OUTPUT_PATHS:-}" ]]; then
  IFS=: read -r -a output_paths <<< "$CI_MINIMAL_OUTPUT_PATHS"
fi

if [[ -n "${CI_MINIMAL_TEST_CVE_PATHS:-}" ]]; then
  IFS=: read -r -a test_cve_paths <<< "$CI_MINIMAL_TEST_CVE_PATHS"
fi

# ---- End bundle configuration --------------------------------------------

check_reproducible=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --check-reproducible)
      check_reproducible=true
      shift
      ;;
    -h|--help)
      echo "Usage: $0 [--check-reproducible]"
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

for command in curl grep jq sha256sum unzip zip zstd; do
  command -v "$command" >/dev/null || { echo "required command not found: $command" >&2; exit 1; }
done

[[ ${#source_members[@]} -eq ${#source_selections[@]} ]] || {
  echo "source_members and source_selections must have the same length" >&2
  exit 1
}

work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT

source_zip=${SOURCE_BUNDLE_ZIP:-/tmp/scanner-v4-dev-vulnerabilities.zip}
if [[ ! -s "$source_zip" ]]; then
  download_path="$work_dir/source.zip"
  curl --fail --retry 3 --connect-timeout 10 --max-time 900 -o "$download_path" "$source_bundle_url"
  mkdir -p "$(dirname "$source_zip")"
  mv "$download_path" "$source_zip"
fi

[[ -s "$source_zip" ]] || { echo "source archive not found: $source_zip" >&2; exit 1; }

if [[ -n "$source_bundle_sha256" ]]; then
  actual_sha256=$(sha256sum "$source_zip" | cut -d' ' -f1)
  [[ "$actual_sha256" == "$source_bundle_sha256" ]] || {
    echo "source SHA256 mismatch: expected $source_bundle_sha256, got $actual_sha256" >&2
    exit 1
  }
fi

source_dir="$work_dir/source"
mkdir -p "$source_dir"
unzip -tq "$source_zip"
listing=$(unzip -Z1 "$source_zip")
for member in "${source_members[@]}"; do
  # Production archives use bundles/; older snapshots use root members.
  archive_member=$(printf '%s\n' "$listing" | grep -Ex "(bundles/)?${member//./\\.}" || true)
  [[ -n "$archive_member" && "$archive_member" != *$'\n'* ]] || {
    echo "missing or ambiguous source member: $member" >&2
    exit 1
  }
  unzip -p "$source_zip" "$archive_member" > "$source_dir/$member"
  [[ -s "$source_dir/$member" ]] || { echo "missing source member: $member" >&2; exit 1; }
done

test_cves="$work_dir/test-cves.txt"
all_cves="$work_dir/all-cves.txt"
grep_matches="$work_dir/grep-matches.txt"
set +e
grep -REho 'CVE-[0-9]{4}-[0-9]+|RH[BS]A-[0-9]{4}:[0-9]+|ALAS[0-9]*-[0-9]{4}-[0-9]+|GO-[0-9]{4}-[0-9]+|GHSA-[a-z0-9-]+' "${test_cve_paths[@]}" > "$grep_matches"
grep_status=$?
set -e
[[ $grep_status -eq 0 || $grep_status -eq 1 ]] || { echo "grep failed: $grep_status" >&2; exit "$grep_status"; }
sort -u "$grep_matches" > "$test_cves"
cp "$test_cves" "$all_cves"
printf '%s\n' "${additional_cves[@]}" >> "$all_cves"
sort -u -o "$all_cves" "$all_cves"

test_cves_json=$(jq -R -s 'split("\n") | map(select(length > 0)) | if length == 0 then {} else map({(.): true}) | add end' "$test_cves")
packages_json=$(printf '%s\n' "${alpine_packages[@]}" | jq -R -s 'split("\n") | map(select(length > 0)) | map({(.): true}) | add')
qa_packages_json=$(jq -e 'select((.distributions | type) == "object" and (.repositories | type) == "object")' "$package_selection")

filter_candidates() {
  local selection=$1 patterns=$2
  if [[ "$selection" == all ]]; then
    cat
  else
    grep -F -f "$patterns" || [[ $? -eq 1 ]]
  fi
}

build_bundle() {
  local bundle_dir=$1
  local output_zip="$bundle_dir/vulnerabilities.zip"
  mkdir -p "$bundle_dir/files"
  cp "$all_cves" "$bundle_dir/ids.txt"

  for index in "${!source_members[@]}"; do
    local archive_member=${source_members[$index]}
    local selection=${source_selections[$index]}
    echo "Selecting $archive_member ($selection)"
    local input="$source_dir/$archive_member"
    local output filtered
    output="$bundle_dir/files/$(basename "$archive_member")"
    filtered="$bundle_dir/$(basename "$archive_member").filtered"
    sort -u "$bundle_dir/ids.txt" | jq -Rn '[inputs | {( . ): true}] | add // {}' > "$bundle_dir/ids.json"
    local patterns="$bundle_dir/candidates.txt"
    cp "$test_cves" "$patterns"
    # Aliases store the namespace and the identifier in separate fields.
    sed 's/^[^-]*-//' "$test_cves" >> "$patterns"
    if [[ "$selection" == enrichment ]]; then
      cp "$bundle_dir/ids.txt" "$patterns"
    elif [[ "$selection" == packages ]]; then
      jq -nr --argjson packages "$qa_packages_json" \
        '$packages | (.distributions[], .repositories[]) | keys[] | "\"name\":" + tojson' >> "$patterns"
    elif [[ "$selection" == alpine ]]; then
      jq -nr --argjson packages "$packages_json" '$packages | keys[] | "\"name\":" + tojson' >> "$patterns"
    fi
    # Validate every record before the conservative text prefilter. Exact
    # identity/package selection below still decides which records to retain.
    zstd -q -dc "$input" | jq -c -L scanner/updater/ci 'include "selection"; validate_record' |
      filter_candidates "$selection" "$patterns" |
      jq -c -L scanner/updater/ci --arg selection "$selection" \
      --argjson test_cves "$test_cves_json" --slurpfile ids "$bundle_dir/ids.json" \
      --argjson packages "$qa_packages_json" --argjson alpine_packages "$packages_json" \
      --arg did "$alpine_distribution_id" --arg version "$alpine_version_id" \
      'include "selection"; select_record($selection; $test_cves; $ids[0]; $packages; $alpine_packages; $did; $version)' > "$filtered"
    if [[ "$archive_member" == rhel-vex.json.zst ]]; then
      # Advisory-only matches also need CVE records for unaffected ranges.
      jq -r -L scanner/updater/ci 'include "selection"; vulnerability_ids' "$filtered" |
        sort -u | jq -Rn '[inputs | {( . ): true}] | add // {}' > "$bundle_dir/rhel-ids.json"
      zstd -q -dc "$input" | jq -c --slurpfile ids "$bundle_dir/rhel-ids.json" \
        'select(.Kind == "vulnerability" and ($ids[0][.Vuln.name] // false))' > "$filtered"
    fi
    if [[ "$selection" != enrichment ]]; then
      jq -r -L scanner/updater/ci 'include "selection"; vulnerability_ids' "$filtered" >> "$bundle_dir/ids.txt"
    fi
    [[ -s "$filtered" ]] || { echo "required selection is empty: $archive_member" >&2; exit 1; }
    ZSTD_CLEVEL=3 zstd -q -3 -T1 -f -o "$output" "$filtered"
    chmod 0644 "$output"
    touch -d '1980-01-01 00:00:00 UTC' "$output"
    rm -f "$filtered"
  done

  (
    cd "$bundle_dir/files"
    LC_ALL=C TZ=UTC ZIPOPT='' zip -X -D -0 -q "$output_zip" ./*.json.zst
  )
}

validate_bundle() {
  local bundle=$1
  unzip -tq "$bundle"
  local listing
  listing=$(unzip -Z1 "$bundle")
  for required_source in "${source_members[@]}"; do
    required_source=${required_source#*/}
    set +e
    printf '%s\n' "$listing" | grep -Fx -- "$required_source" >/dev/null
    local grep_status=$?
    set -e
    [[ $grep_status -eq 0 ]] || {
      echo "generated bundle is missing required source: $required_source" >&2
      return 1
    }
  done
  set +e
  printf '%s\n' "$listing" | grep -q 'synthetic'
  local synthetic_status=$?
  set -e
  if [[ $synthetic_status -eq 0 ]]; then
    echo "generated bundle contains a synthetic source" >&2
    return 1
  elif [[ $synthetic_status -ne 1 ]]; then
    echo "grep failed while validating bundle members: $synthetic_status" >&2
    return "$synthetic_status"
  fi

  local nvd_file="$work_dir/validate-nvd.jsonl"
  local alpine_file="$work_dir/validate-alpine.jsonl"
  unzip -p "$bundle" nvd.json.zst | zstd -q -dc > "$nvd_file"
  unzip -p "$bundle" alpine.json.zst | zstd -q -dc > "$alpine_file"
  jq -e -s --argjson additional "$(printf '%s\n' "${additional_cves[@]}" | jq -R -s 'split("\n") | map(select(length > 0)) | map({(.): true}) | add')" \
    '([.[] | .Enrichment.Enrichment.id] | map(select(. != null))) as $ids |
     all($additional | keys[]; . as $id | ($ids | index($id)) != null)' "$nvd_file" >/dev/null
  jq -e -s 'any(.[]; .Vuln.package.name == "nginx" and .Vuln.distribution.did == "alpine" and
    .Vuln.distribution.version_id == "3.9" and (.Vuln.fixed_in_version // "") != "")' "$alpine_file" >/dev/null
  jq -e -s --slurpfile nvd "$nvd_file" '
    any(.[]; .Vuln as $v |
      $v.package.name == "nginx" and $v.distribution.did == "alpine" and
      $v.distribution.version_id == "3.9" and ($v.fixed_in_version // "") != "" and
      ([ $nvd[] | select(.Enrichment.Enrichment.id == $v.name) ] | length > 0))' "$alpine_file" >/dev/null

  local matched_ids="$work_dir/matched-ids.txt"
  : > "$matched_ids"
  for index in "${!source_members[@]}"; do
    [[ "${source_selections[$index]}" != enrichment ]] || continue
    unzip -p "$bundle" "${source_members[$index]}" | zstd -q -dc |
      jq -r -L scanner/updater/ci 'include "selection"; vulnerability_ids' >> "$matched_ids"
  done
  for cve in "${qa_cves[@]}"; do
    # Fixture tests may supply a different test corpus.
    if grep -Fxq "$cve" "$test_cves"; then
      grep -Fxq "$cve" "$matched_ids" || { echo "missing QA vulnerability: $cve" >&2; return 1; }
      jq -e -s --arg cve "$cve" 'any(.[]; .Enrichment.Enrichment.id == $cve)' "$nvd_file" >/dev/null || {
        echo "missing QA enrichment: $cve" >&2
        return 1
      }
    fi
  done
}

if "$check_reproducible"; then
  build_bundle "$work_dir/bundle-one"
  build_bundle "$work_dir/bundle-two"
  cmp "$work_dir/bundle-one/vulnerabilities.zip" "$work_dir/bundle-two/vulnerabilities.zip"
  echo "Reproducibility check passed: $(sha256sum "$work_dir/bundle-one/vulnerabilities.zip" | cut -d' ' -f1)"
  generated_zip="$work_dir/bundle-one/vulnerabilities.zip"
else
  build_bundle "$work_dir/bundle"
  generated_zip="$work_dir/bundle/vulnerabilities.zip"
fi

validate_bundle "$generated_zip"

staged_paths=()
backup_paths=()
published_paths=()
rollback_publication() {
  local i path backup
  for path in "${published_paths[@]}"; do
    rm -f "$path"
  done
  for ((i=${#backup_paths[@]}-1; i>=0; i--)); do
    backup=${backup_paths[$i]}
    [[ -e "$backup" ]] && mv "$backup" "${output_paths[$i]}"
  done
  for path in "${staged_paths[@]}"; do
    rm -f "$path"
  done
}

publish_failed=false
for i in "${!output_paths[@]}"; do
  output_path=${output_paths[$i]}
  mkdir -p "$(dirname "$output_path")"
  staged=$(mktemp "$(dirname "$output_path")/.$(basename "$output_path").tmp.XXXXXX")
  cp "$generated_zip" "$staged"
  staged_paths[i]=$staged
  backup_paths[i]="$output_path.bak.$$.$i"
done
for i in "${!output_paths[@]}"; do
  output_path=${output_paths[$i]}
  backup=${backup_paths[$i]}
  if [[ -e "$output_path" && ! -e "$backup" ]] && ! mv "$output_path" "$backup"; then
    publish_failed=true
    break
  fi
  if [[ "${CI_MINIMAL_PUBLISH_FAIL_AFTER:-0}" -eq $((i + 1)) ]]; then
    publish_failed=true
    break
  fi
  if ! mv "${staged_paths[$i]}" "$output_path"; then
    publish_failed=true
    break
  fi
  published_paths+=("$output_path")
done
if "$publish_failed"; then
  rollback_publication
  echo "publishing generated bundle failed" >&2
  exit 1
fi
rm -f "${backup_paths[@]}" "${staged_paths[@]}"

echo "Generated CI-minimal bundle: $(sha256sum "$generated_zip" | cut -d' ' -f1)"
unzip -l "$generated_zip"

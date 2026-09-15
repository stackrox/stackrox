#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/../../.." && pwd)
cd "$repo_root"

# ---- Bundle configuration -------------------------------------------------

source_bundle_url="${SOURCE_BUNDLE_URL:-https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulnerabilities.zip}"
# Set this when the mutable development URL is pinned to a saved archive.
source_bundle_sha256=""

source_members=(
  "alpine.json.zst"
  "debian.json.zst"
  "nvd.json.zst"
)
source_selections=(
  alpine
  test-cves
  all-cves
)

test_cve_paths=(
  "scanner/e2etests/testdata"
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

for command in curl jq rg sha256sum unzip zip zstd; do
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
unzip -q "$source_zip" "${source_members[@]}" -d "$source_dir"
for member in "${source_members[@]}"; do
  [[ -s "$source_dir/$member" ]] || { echo "missing source member: $member" >&2; exit 1; }
done

test_cves="$work_dir/test-cves.txt"
all_cves="$work_dir/all-cves.txt"
rg_matches="$work_dir/rg-matches.txt"
set +e
rg --no-filename -o 'CVE-[0-9]{4}-[0-9]+' "${test_cve_paths[@]}" > "$rg_matches"
rg_status=$?
set -e
[[ $rg_status -eq 0 || $rg_status -eq 1 ]] || { echo "rg failed: $rg_status" >&2; exit "$rg_status"; }
sort -u "$rg_matches" > "$test_cves"
cp "$test_cves" "$all_cves"
printf '%s\n' "${additional_cves[@]}" >> "$all_cves"
sort -u -o "$all_cves" "$all_cves"

cves_json=$(jq -R -s 'split("\n") | map(select(length > 0)) | if length == 0 then {} else map({(.): true}) | add end' "$all_cves")
test_cves_json=$(jq -R -s 'split("\n") | map(select(length > 0)) | if length == 0 then {} else map({(.): true}) | add end' "$test_cves")
packages_json=$(printf '%s\n' "${alpine_packages[@]}" | jq -R -s 'split("\n") | map(select(length > 0)) | map({(.): true}) | add')

build_bundle() {
  local bundle_dir=$1
  local output_zip="$bundle_dir/vulnerabilities.zip"
  mkdir -p "$bundle_dir/files"

  for index in "${!source_members[@]}"; do
    local archive_member=${source_members[$index]}
    local selection=${source_selections[$index]}
    local input="$source_dir/$archive_member"
    local output raw filtered
    output="$bundle_dir/files/$(basename "$archive_member")"
    raw="$bundle_dir/$(basename "$archive_member").raw"
    filtered="$bundle_dir/$(basename "$archive_member").filtered"
    zstd -q -dc "$input" > "$raw"
    jq -e -c 'select(
      type == "object" and (.Kind == "vulnerability" or .Kind == "enrichment") and
      (.Updater | type == "string") and (.Fingerprint | type == "string") and
      (.Ref | type == "string") and (.Date | type == "string") and
      ((.Kind == "vulnerability" and (.Vuln | type == "object")) or
       (.Kind == "enrichment" and (.Enrichment | type == "object") and
        (.Enrichment.Enrichment | type == "object") and
        (.Enrichment.Enrichment.id | type == "string"))))' "$raw" > "$bundle_dir/validated.jsonl"
    [[ -s "$bundle_dir/validated.jsonl" ]] || {
      echo "source member is empty or has no valid importer records: $archive_member" >&2
      exit 1
    }
    case "$selection" in
      alpine)
        jq -c --argjson test_cves "$test_cves_json" --argjson packages "$packages_json" \
            --arg did "$alpine_distribution_id" --arg version "$alpine_version_id" \
            'select(.Kind == "vulnerability") | .Vuln as $v | ($v.distribution // {}) as $d | ($v.package // {}) as $p |
              select(($test_cves[$v.name // ""] // false) or
                ($d.did == $did and $d.version_id == $version and ($packages[$p.name // ""] // false)))' \
            "$bundle_dir/validated.jsonl" > "$filtered"
        ;;
      test-cves)
        jq -c --argjson cves "$test_cves_json" \
          'select(.Kind == "vulnerability") | .Vuln as $v | select($cves[$v.name // ""] // false)' \
          "$bundle_dir/validated.jsonl" > "$filtered"
        ;;
      all-cves)
        jq -c --argjson cves "$cves_json" \
          'select(.Kind == "enrichment") | .Enrichment.Enrichment as $en | select($cves[$en.id // ""] // false)' \
          "$bundle_dir/validated.jsonl" > "$filtered"
        ;;
      *)
        echo "unsupported selection '$selection' for source '$archive_member'" >&2
        exit 1
        ;;
    esac
    [[ -s "$filtered" ]] || { echo "required selection is empty: $archive_member" >&2; exit 1; }
    ZSTD_CLEVEL=3 zstd -q -3 -T1 -f -o "$output" "$filtered"
    chmod 0644 "$output"
    touch -d '1980-01-01 00:00:00 UTC' "$output"
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
    printf '%s\n' "$listing" | rg -Fx -- "$required_source" >/dev/null
    local rg_status=$?
    set -e
    [[ $rg_status -eq 0 ]] || {
      echo "generated bundle is missing required source: $required_source" >&2
      return 1
    }
  done
  set +e
  printf '%s\n' "$listing" | rg -q 'synthetic'
  local synthetic_status=$?
  set -e
  if [[ $synthetic_status -eq 0 ]]; then
    echo "generated bundle contains a synthetic source" >&2
    return 1
  elif [[ $synthetic_status -ne 1 ]]; then
    echo "rg failed while validating bundle members: $synthetic_status" >&2
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

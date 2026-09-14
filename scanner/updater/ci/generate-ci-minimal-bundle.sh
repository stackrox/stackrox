#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/../../.." && pwd)
cd "$repo_root"

# ---- Bundle configuration -------------------------------------------------

source_bundle_url="https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulnerabilities.zip"
# Set this when the mutable development URL is pinned to a saved archive.
source_bundle_sha256=""

source_members=(
  "bundles/alpine.json.zst"
  "bundles/debian.json.zst"
  "bundles/nvd.json.zst"
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

if [[ -n "${SOURCE_BUNDLE_ZIP:-}" ]]; then
  source_zip=$SOURCE_BUNDLE_ZIP
  [[ -s "$source_zip" ]] || { echo "source archive not found: $source_zip" >&2; exit 1; }
else
  source_zip="$work_dir/source.zip"
  curl --fail --retry 3 --connect-timeout 10 --max-time 900 -o "$source_zip" "$source_bundle_url"
fi

if [[ -n "$source_bundle_sha256" ]]; then
  actual_sha256=$(sha256sum "$source_zip" | cut -d' ' -f1)
  [[ "$actual_sha256" == "$source_bundle_sha256" ]] || {
    echo "source SHA256 mismatch: expected $source_bundle_sha256, got $actual_sha256" >&2
    exit 1
  }
fi

source_dir="$work_dir/source"
mkdir -p "$source_dir"
unzip -q "$source_zip" "${source_members[@]}" -d "$source_dir"
for member in "${source_members[@]}"; do
  [[ -s "$source_dir/$member" ]] || { echo "missing source member: $member" >&2; exit 1; }
done

test_cves="$work_dir/test-cves.txt"
all_cves="$work_dir/all-cves.txt"
set +e
rg --no-filename -o 'CVE-[0-9]{4}-[0-9]+' "${test_cve_paths[@]}" | sort -u > "$test_cves"
rg_status=$?
set -e
[[ $rg_status -eq 0 || $rg_status -eq 1 ]] || exit "$rg_status"
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
    archive_member=${source_members[$index]}
    selection=${source_selections[$index]}
    local input="$source_dir/$archive_member"
    local output
    output="$bundle_dir/files/$(basename "$archive_member")"
    case "$selection" in
      alpine)
        zstd -q -T1 -dc "$input" \
          | jq -c --argjson test_cves "$test_cves_json" --argjson packages "$packages_json" \
              --arg did "$alpine_distribution_id" --arg version "$alpine_version_id" \
              '(.Vuln // {}) as $v | (($v.distribution // {}) as $d | ($v.package // {}) as $p |
                select(($test_cves[$v.name // ""] // false) or
                  ($d.did == $did and $d.version_id == $version and ($packages[$p.name // ""] // false))))' \
          | zstd -q -T1 -o "$output"
        ;;
      test-cves)
        zstd -q -T1 -dc "$input" \
          | jq -c --argjson cves "$test_cves_json" '(.Vuln // {}) as $v | select($cves[$v.name // ""] // false)' \
          | zstd -q -T1 -o "$output"
        ;;
      all-cves)
        zstd -q -T1 -dc "$input" \
          | jq -c --argjson cves "$cves_json" '(.Enrichment // {}) as $e | ($e.Enrichment // {}) as $en | select($cves[$en.id // ""] // false)' \
          | zstd -q -T1 -o "$output"
        ;;
      *)
        echo "unsupported selection '$selection' for source '$archive_member'" >&2
        exit 1
        ;;
    esac
    chmod 0644 "$output"
    touch -d '1980-01-01 00:00:00 UTC' "$output"
  done

  (
    cd "$bundle_dir/files"
    LC_ALL=C TZ=UTC zip -X -D -q "$output_zip" ./*.json.zst
  )
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

for output_path in "${output_paths[@]}"; do
  mkdir -p "$(dirname "$output_path")"
  cp "$generated_zip" "$output_path"
done

for required_source in "${source_members[@]}"; do
  required_source=${required_source#*/}
  unzip -Z1 "$generated_zip" | rg -Fx -- "$required_source" >/dev/null || {
    echo "generated bundle is missing required source: $required_source" >&2
    exit 1
  }
done
if unzip -Z1 "$generated_zip" | rg -q 'synthetic'; then
  echo "generated bundle contains a synthetic source" >&2
  exit 1
fi

echo "Generated CI-minimal bundle: $(sha256sum "$generated_zip" | cut -d' ' -f1)"
unzip -l "$generated_zip"

#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/../../.." && pwd)
cd "$repo_root"

source_zip=${SOURCE_BUNDLE_ZIP:-/tmp/scanner-v4-dev-vulnerabilities.zip}
source_url=${SOURCE_BUNDLE_URL:-https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulnerabilities.zip}
work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT

if [[ ! -s "$source_zip" ]]; then
  curl --fail --retry 3 --connect-timeout 10 --max-time 900 -o "$source_zip" "$source_url"
fi

unzip -q "$source_zip" \
  'bundles/alpine.json.zst' 'bundles/debian.json.zst' 'bundles/nvd.json.zst' \
  -d "$work_dir/source"

# Keep every CVE referenced by the CI tests. The nginx image is intentionally
# selected independently because its old Alpine 3.9 package set is not covered
# by the generic CVE allowlist.
rg --no-filename -o 'CVE-[0-9]{4}-[0-9]+' scanner/e2etests/testdata qa-tests-backend/src/test/groovy \
  | sort -u > "$work_dir/cves.txt"
cp "$work_dir/cves.txt" "$work_dir/test-cves.txt"
cat >> "$work_dir/cves.txt" <<'EOF'
CVE-2017-7529
CVE-2018-16843
CVE-2018-16844
CVE-2018-16845
CVE-2019-9511
CVE-2019-9513
CVE-2019-9516
CVE-2019-20372
EOF
sort -u -o "$work_dir/cves.txt" "$work_dir/cves.txt"

cat > "$work_dir/image-packages.txt" <<'EOF'
busybox
freetype
libgcrypt
libjpeg-turbo
libpng
libxml2
libxslt
musl
pcre
EOF

cves_json=$(jq -R -s 'split("\n") | map(select(length > 0)) | map({(.): true}) | add' "$work_dir/cves.txt")
test_cves_json=$(jq -R -s 'split("\n") | map(select(length > 0)) | map({(.): true}) | add' "$work_dir/test-cves.txt")
packages_json=$(jq -R -s 'split("\n") | map(select(length > 0)) | map({(.): true}) | add' "$work_dir/image-packages.txt")
mkdir -p "$work_dir/bundles"

zstd -q -dc "$work_dir/source/bundles/alpine.json.zst" \
  | jq -c --argjson test_cves "$test_cves_json" --argjson packages "$packages_json" \
      '(.Vuln // {}) as $v | (($v.distribution // {}) as $d | ($v.package // {}) as $p |
        select(($test_cves[$v.name // ""] // false) or
              ($d.did == "alpine" and $d.version_id == "3.9" and
               (($packages[$p.name // ""] // false) or $p.name == "nginx"))))' \
  | zstd -q -T0 -o "$work_dir/bundles/alpine.json.zst"

zstd -q -dc "$work_dir/source/bundles/debian.json.zst" \
  | jq -c --argjson cves "$test_cves_json" '(.Vuln // {}) as $v | select($cves[$v.name // ""] // false)' \
  | zstd -q -T0 -o "$work_dir/bundles/debian.json.zst"

zstd -q -dc "$work_dir/source/bundles/nvd.json.zst" \
  | jq -c --argjson cves "$cves_json" \
      '(.Enrichment // {}) as $e | ($e.Enrichment // {}) as $en |
       select($cves[$en.id // ""] // false)' \
  | zstd -q -T0 -o "$work_dir/bundles/nvd.json.zst"

mkdir -p scanner/updater/ci/bundles/ci-minimal scanner/image/scanner/bundles/ci-minimal
rm -f scanner/updater/ci/bundles/ci-minimal/*.json.zst scanner/updater/ci/bundles/ci-minimal/vulnerabilities.zip
cp "$work_dir/bundles"/*.json.zst scanner/updater/ci/bundles/ci-minimal/
(
  cd scanner/updater/ci/bundles/ci-minimal
  zip -X -q vulnerabilities.zip *.json.zst
)
cp scanner/updater/ci/bundles/ci-minimal/vulnerabilities.zip \
  scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip

echo "Generated ci-minimal bundle from $source_url"
unzip -l scanner/updater/ci/bundles/ci-minimal/vulnerabilities.zip

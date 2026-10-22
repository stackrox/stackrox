#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
# shellcheck source=../../../scripts/ci/scanner-bundle.sh
source "$root/scripts/ci/scanner-bundle.sh"

[[ "${GITHUB_REF:?}" == refs/heads/master ]] || { echo 'Only master may publish CI bundles' >&2; exit 1; }
bundle="$root/scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip"
version="$(scanner_v4_ci_bundle_version)"
prefix="gs://definitions.stackrox.io/v4/vulnerability-bundles/$version"
object="$prefix/vulnerabilities.zip"
url="https://definitions.stackrox.io/v4/vulnerability-bundles/$version/vulnerabilities.zip"
workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT

# A failed listing is an error, not evidence that the object is absent.
existing="$(gcloud storage objects list "$prefix/*" --format='value(name)')"
if [[ -z "$existing" ]]; then
    gcloud storage cp "$bundle" "$object" --if-generation-match=0
fi
# Verify existing objects too, without rewriting their Last-Modified timestamp.
gcloud storage cp "$object" "$workdir/stored.zip"
cmp "$bundle" "$workdir/stored.zip"
curl --fail --silent --show-error --location --dump-header "$workdir/headers" --output "$workdir/public.zip" "$url"
cmp "$bundle" "$workdir/public.zip"
modified="$(python3 - "$workdir/headers" <<'PY'
import email.utils
import pathlib
import sys

headers = pathlib.Path(sys.argv[1]).read_text().splitlines()
values = [line.split(':', 1)[1].strip() for line in headers if line.lower().startswith('last-modified:')]
if not values:
    sys.exit('Missing Last-Modified header')
value = values[-1]
timestamp = email.utils.parsedate_to_datetime(value)
if timestamp.tzinfo is None:
    sys.exit('Invalid Last-Modified header')
print(value)
PY
)"
status="$(curl --fail --silent --show-error --location --header "If-Modified-Since: $modified" --output /dev/null --write-out '%{http_code}' "$url")"
[[ "$status" == 304 ]] || { echo "Expected HTTP 304, got $status" >&2; exit 1; }
printf 'Published Scanner CI bundle: %s\n%s\n' "$version" "$url" | tee -a "${GITHUB_STEP_SUMMARY:-/dev/null}"

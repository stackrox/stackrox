#!/usr/bin/env bash
# Verifies that a given release version already exists in
# scanner/updater/version/RELEASE_VERSION on the master branch.
#
# Usage (locally):
#   bash local-env.sh check-scanner-version <version>

set -euo pipefail

TAG="$1"
check_not_empty TAG
VERSION="${TAG/-rc.[0-9]*/}"

# Fetch the file content from the master branch (raw).
SCANNER_VERSION=$(gh api -H "Accept: application/vnd.github.v3.raw" \
  "/repos/${GITHUB_REPOSITORY}/contents/scanner/updater/version/RELEASE_VERSION?ref=master")

if ! grep -q "^${VERSION}$" <<<"$SCANNER_VERSION"; then
    gh_log error "Release version $VERSION (inferred from the tag '$TAG') not added to scanner/updater/version/RELEASE_VERSION in master branch"
    gh_summary "Release version not found in scanner/updater/version/RELEASE_VERSION in master branch"
    gh_summary "Most likely, this is because the PR to update scanner version file created by \`start-release\` workflow is not merged"
    gh_summary "➡️  Please check the PR created by \`start-release\` workflow that started this release."
    gh_summary "➡️  There should be $VERSION in the \`RELEASE_VERSION\` file in master."
    exit 1
fi
gh_summary "✅ Version ${VERSION} present in scanner/updater/version/RELEASE_VERSION."

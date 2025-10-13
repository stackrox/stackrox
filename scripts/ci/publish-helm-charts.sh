#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

[[ -n "${GITHUB_TOKEN}" ]] || die "No GitHub token found"

remote_repository="https://github.com/stackrox/release-artifacts.git"
remote_subdirectory="helm-charts"

version="$1"
central_services_chart="$2"
secured_cluster_services_chart="$3"

[[ -n "$version" ]] || die "No version specified"
[[ -n "$central_services_chart" ]] || die "No central services chart path specified!"
[[ -n "$secured_cluster_services_chart" ]] || die "No secured cluster services chart path specified!"

echo "Publishing charts for version $version"
echo " Central Services Chart location: ${central_services_chart}"
echo " Secured Cluster Services Chart location: ${secured_cluster_services_chart}"

if is_release_test_stream "$version"; then
	# send to #acs-slack-integration-testing when testing the release process
	webhook_url="${SLACK_MAIN_WEBHOOK}"
else
	# send to #acs-release-notifications
	webhook_url="${RELEASE_WORKFLOW_NOTIFY_WEBHOOK}"
fi

[[ -n "${webhook_url}" ]] || die "No Slack webhook found"

tmp_remote_repository="$(mktemp -d)"

gitbot clone "$remote_repository" "$tmp_remote_repository"

branch_name="release/${version}"

gitbot -C "$tmp_remote_repository" checkout -b "$branch_name"

mkdir "${tmp_remote_repository}/${remote_subdirectory}/${version}"

cp -a "${central_services_chart}/opensource" "${tmp_remote_repository}/${remote_subdirectory}/${version}/central-services"
cp -a "${secured_cluster_services_chart}/opensource" "${tmp_remote_repository}/${remote_subdirectory}/${version}/secured-cluster-services"

mkdir "${tmp_remote_repository}/${remote_subdirectory}/rhacs/${version}"

cp -a "${central_services_chart}/rhacs" "${tmp_remote_repository}/${remote_subdirectory}/rhacs/${version}/central-services"
cp -a "${secured_cluster_services_chart}/rhacs" "${tmp_remote_repository}/${remote_subdirectory}/rhacs/${version}/secured-cluster-services"

mkdir -p "${tmp_remote_repository}/${remote_subdirectory}/opensource"

echo "Packaging Helm chart for file ${central_services_chart}/opensource/Chart.yaml"
helm package -d "${tmp_remote_repository}/${remote_subdirectory}/opensource" "${central_services_chart}/opensource"
echo "Packaging Helm chart for file ${secured_cluster_services_chart}/opensource/Chart.yaml"
helm package -d "${tmp_remote_repository}/${remote_subdirectory}/opensource" "${secured_cluster_services_chart}/opensource"

echo "Building OSS helm repo index"
helm repo index "${tmp_remote_repository}/${remote_subdirectory}/opensource"

echo "Adding Artifact Hub meta info file"
cp "${ROOT}/scripts/ci/artifacthub/artifacthub-repo.yml" "${tmp_remote_repository}/${remote_subdirectory}/opensource/artifacthub-repo.yml"

gitbot -C "$tmp_remote_repository" add -A
gitbot -C "$tmp_remote_repository" commit -m "Publish Helm Charts for version ${version}"
gitbot -C "$tmp_remote_repository" push origin "$branch_name"

message="Hello Release Managers!

Engineering has signed off on release **${version}**! The new Helm charts are ready to be published.
Once the version is GA, please complete the publishing by merging this PR using the 'Squash and Merge' option.

**CAUTION**: If this PR is merged prior to the GA announcement, customers might accidentally and unknowingly upgrade
to an unreleased version."

curl -sS --fail \
	-X POST \
	-H "Authorization: token ${GITHUB_TOKEN}" \
	'https://api.github.com/repos/stackrox/release-artifacts/pulls' \
	-d"{
	\"title\": \"Publishing release artifacts for release ${version}\",
	\"body\": $(jq -sR <<<"$message"),
	\"head\": \"${branch_name}\",
	\"base\": \"main\"
}" || die "Failed to create GitHub PR"

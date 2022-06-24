#!/usr/bin/env bash

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

[[ -n "${GITHUB_TOKEN}" ]] || die "No GitHub token found"

user_name='roxbot'
user_email='roxbot@stackrox.com'

remote_repository="git@github.com:stackrox/release-artifacts.git"
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
	# send to #slack-test when testing the release process
	webhook_url="${SLACK_MAIN_WEBHOOK}"
else
	# send to #eng-release
	webhook_url="${RELEASE_WORKFLOW_NOTIFY_WEBHOOK}"
fi

[[ -n "${webhook_url}" ]] || die "No Slack webhook found"

tmp_remote_repository="$(mktemp -d)"

git clone "$remote_repository" "$tmp_remote_repository"

branch_name="release/${version}"

git -C "$tmp_remote_repository" checkout -b "$branch_name"

mkdir "${tmp_remote_repository}/${remote_subdirectory}/${version}"

cp -a "${central_services_chart}/stackrox" "${tmp_remote_repository}/${remote_subdirectory}/${version}/central-services"
cp -a "${secured_cluster_services_chart}/stackrox" "${tmp_remote_repository}/${remote_subdirectory}/${version}/secured-cluster-services"

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

git -C "$tmp_remote_repository" add -A
git -C "$tmp_remote_repository" -c "user.name=${user_name}" -c "user.email=${user_email}" commit -m "Publish Helm Charts for version ${version}"
git -C "$tmp_remote_repository" push origin "$branch_name"

pr_response_file="$(mktemp)"

message="Hello Release Artifact Publishers!

Engineering has signed off on release **${version}**! The new Helm charts are ready to be published.
Once the version is GA, please complete the publishing by merging this PR using the 'Squash and Merge' option.

**CAUTION**: If this PR is merged prior to the GA announcement, customers might accidentally and unknowingly upgrade
to an unreleased version."

curl -sS --fail \
	-o "$pr_response_file" \
	-X POST \
	-H "Authorization: token ${GITHUB_TOKEN}" \
	'https://api.github.com/repos/stackrox/release-artifacts/pulls' \
	-d"{
	\"title\": \"Publishing release artifacts for release ${version}\",
	\"body\": $(jq -sR <<<"$message"),
	\"head\": \"${branch_name}\",
	\"base\": \"main\"
}" || die "Failed to create GitHub PR"

pr_number="$(jq <"$pr_response_file" -r '.number')"

[[ -n "$pr_number" ]] || die "Failed to determine PR number"

curl -sS --fail \
	-X POST \
	-H "Authorization: token ${GITHUB_TOKEN}" \
	"https://api.github.com/repos/stackrox/release-artifacts/pulls/${pr_number}/requested_reviewers" \
	-d'{
	"team_reviewers": ["release-artifact-publishers"]
}' || die "Failed to assign release-artifact-publishers for review"

jq -n \
    --arg version "$version" \
    --arg pr_number "$pr_number" \
    '{"text": "Hey <!subteam^S01DE67NT7V|release-publishers>! A pull request for the *\($version)* release artifacts has been prepared. Once this version is GA, please _approve and merge_ this pull request in order to publish the artifacts: https://github.com/stackrox/release-artifacts/pull/\($pr_number)"}' \
  | curl -XPOST -d @- -H 'Content-Type: application/json' "${webhook_url}" || die "Failed to send Slack message!"

#!/usr/bin/env bash

workdir="$(mktemp -d)"

echo "Workdir is ${workdir}"

git -C "$workdir" clone git@github.com:stackrox/rox.git .

for tag in $(git -C "$workdir" tag | egrep '^3\.(6[3-9]|0\.62)\.\d+$' | sort -V); do
	version="$(echo "$tag" | sed -E 's@^3.0.([[:digit:]]+\.[[:digit:]]+)(-)?@3.\1\2@g')"
	docker pull "docker.io/stackrox/stackrox-operator:${version}"
	docker tag "docker.io/stackrox/stackrox-operator:${version}" "quay.io/rhacs-eng/stackrox-operator:${version}"
	docker push "quay.io/rhacs-eng/stackrox-operator:${version}"

	git -C "$workdir" checkout "$tag"
	CI=1 BUILD_TAG="$tag" IMAGE_REPO="quay.io/rhacs-eng" IMAGE_TAG_BASE="quay.io/rhacs-eng/stackrox-operator" make -C "$workdir/operator" bundle-build
	docker push "quay.io/rhacs-eng/stackrox-operator-bundle:v${version}"

	CI=1 BUILD_TAG="$tag" IMAGE_REPO="quay.io/rhacs-eng" IMAGE_TAG_BASE="quay.io/rhacs-eng/stackrox-operator" make -C "$workdir/operator" index-build
	docker push "quay.io/rhacs-eng/stackrox-operator-index:v${version}"
done

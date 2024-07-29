#!/usr/bin/env bash

# This script is in charge of generating the metadata and manifests
# needed to build the operator-bundle on konflux.

set -euxo pipefail

mkdir -p build
rm -rf build/bundle
cp -a bundle build

INDEX_IMG_BASE="quay.io/rhacs-eng/stackrox-operator-index"
IMG="$INDEX_IMG_BASE:v$VERSION"

# Create a virtual environment and install required dependencies
python3 -m venv bundle_helpers/.venv
source bundle_helpers/.venv/bin/activate
PIP_CONSTRAINTS=bundle_helpers/constraints.txt pip3 install --upgrade pip==21.3.1 setuptools==59.6.0
PIP_CONSTRAINTS=bundle_helpers/constraints.txt pip3 install -r bundle_helpers/requirements.txt

first_version=3.62.0 # this is the first operator version ever released

candidate_version=$(./bundle_helpers/patch-csv.py \
    --use-version "$VERSION" \
    --first-version "$first_version" \
    --operator-image "$IMG" \
    --echo-replaced-version-only \
    < bundle/manifests/rhacs-operator.clusterserviceversion.yaml
)

unreleased_opt=""
if ! ../scripts/ci/lib.sh check_rhacs_eng_image_exists "$INDEX_IMG_BASE" v"$candidate_version"; then
    unreleased_opt="--unreleased=$candidate_version"
fi

./bundle_helpers/patch-csv.py \
    --use-version "$VERSION" \
    --first-version "$first_version" \
    --operator-image "$IMG" \
    "$unreleased_opt" \
    --no-related-images \
    --add-supported-arch amd64 \
    --add-supported-arch s390x \
    --add-supported-arch ppc64le \
    --add-supported-arch arm64 \
    < bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
    > build/bundle/manifests/rhacs-operator-operator.clusterserviceversion.yaml

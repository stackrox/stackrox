#!/usr/bin/env bash
# Temporary script to install 4.10 operator to test upgrades to latest version using helm chart.
# Operator is not supported downstream on non-OpenShift platforms before 4.11, but this lets us roughly test
# the upgrade path from the upstream tech preview in 4.10 to the helm chart.
# TODO(ROX-33128): delete and just use helm after release 4.11 ships
set -euo pipefail

version="4.10.0"

dir="$(mktemp -d)"
echo >&2 "Deploying operator version ${version} from a temporary checkout at ${dir}"

git worktree add "${dir}" "${version}"
trap 'git worktree remove --force "${dir}"' EXIT

cd "${dir}"
export VERSION="${version}" ROX_PRODUCT_BRANDING=RHACS_BRANDING
make -C operator/ build-installer deploy-via-installer TEST_NAMESPACE="rhacs-operator-system"

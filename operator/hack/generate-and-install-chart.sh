#!/usr/bin/env bash
# Install previous operator version to test upgrades to latest version using helm chart.
set -euo pipefail

if [[ -z ${1:-} ]]; then
    echo >&2 "Usage: $0 <version>"
    echo >&2 "Example: $0 4.11.2"
    echo >&2 "Note: <version> may optionally include the leading 'v'"
    exit 1
fi

# Strip optional leading 'v'.
version="${1#v}"

# The rhacs-eng chart is not preserved anywhere on releases, so we produce it on the fly.
dir="$(mktemp -d)"
echo >&2 "Deploying operator version ${version} from a temporary checkout at ${dir}"

git worktree add "${dir}" "${version}"
trap 'git worktree remove --force "${dir}"' EXIT

# Warm up go module cache so that transient network failures do not abort controller-gen.
# TODO: this command can be removed once the version we upgrade from runs go-mod-download internally.
(cd "${dir}"; go mod download || go mod download || go mod download)
make -C "${dir}/operator" chart deploy-via-chart VERSION="${version}"

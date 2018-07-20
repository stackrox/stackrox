#!/bin/bash

function usage() {
  echo "To go get a package at a particular version, use:"
  echo "$0 <package> <hash_or_tag> [--skip-install]"
  exit 1
}

[[ "$#" -ge 2 ]] || usage

package="$1"
hash_or_tag="$2"
skip_install="$3"

package_without_trailing_dots="${package%/...}"
go get -d -v "${package_without_trailing_dots}"
cd "${GOPATH}/src/${package_without_trailing_dots}" || { echo "Couldn't cd to the directory!"; exit 1; }

git rev-parse --git-dir > /dev/null 2>&1 || { echo "This script only supports git-based packages!"; exit 1; }
git checkout -q "${hash_or_tag}" || { echo "git checkout failed!"; exit 1; }
[[ "$skip_install" = "--skip-install" ]] || go install "${package}" || { echo "go install failed!"; exit 1; }

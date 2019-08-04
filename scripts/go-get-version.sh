#!/bin/bash

function usage() {
  echo "To go get a package at a particular version, use:"
  echo "$0 <package> <hash_or_tag> [--skip-install]"
  exit 1
}

[[ "$#" -ge 2 ]] || usage

export GO111MODULE=off  # force legacy go get mode

function install() {
  package="$1"
  hash_or_tag="$2"
  skip_install="$3"

  package_without_trailing_dots="${package%/...}"
  go get -d -v "${package_without_trailing_dots}"
  cd "${GOPATH}/src/${package_without_trailing_dots}" || { echo "Couldn't cd to the directory!"; return 1; }

  git rev-parse --git-dir > /dev/null 2>&1 || { echo "This script only supports git-based packages!"; return 1; }
  git checkout -q "${hash_or_tag}" || { echo "git checkout failed!"; exit 1; }
  [[ "$skip_install" = "--skip-install" ]] || go install "${package}" || { echo "go install failed!"; return 1; }
}

for i in {1..10}; do
  if install $@; then
    exit 0
  fi
  sleep 1
done

echo "failed all retries"
exit 1
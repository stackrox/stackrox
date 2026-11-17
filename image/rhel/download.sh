#!/usr/bin/env bash

set -euo pipefail

arch=$(uname -m)
postgres_major="${PG_VERSION:-16}"
goarch="$arch"
if [[ "$arch" == "x86_64" ]]; then
    goarch="amd64"
elif [[ "$arch" == "aarch64" ]]; then
    goarch="arm64"
elif [[ "$arch" == "arm64" ]]; then
    arch="aarch64"
fi

output_dir="/output"
mkdir -p "${output_dir}/go/bin"
if [[ "$DEBUG_BUILD" == "yes" ]]; then
  if [[ "$goarch" == "amd64" || "$goarch" == "arm64" ]]; then
    dnf install -y golang
    if [[ "$OSTYPE" != "linux-gnu"* ]]; then
      GOBIN='' GOOS=linux GOARCH="${goarch}" GOPATH="${output_dir}/go" go install github.com/go-delve/delve/cmd/dlv@latest
      mv "${output_dir}/go/bin/linux_${goarch}/dlv" "${output_dir}/go/bin/dlv"
      rm -r "${output_dir}/go/bin/linux_${goarch}"
    else
      GOBIN="${output_dir}/go/bin" go install github.com/go-delve/delve/cmd/dlv@latest
    fi
  else
    echo "WARNING: Architecture ${goarch} is not supported by delve. Debugging won't be available"
  fi
fi

mkdir -p "$output_dir/rpms"

if [[ "$arch" == "s390x" ]]; then
  rpm --import /tmp/RPM-GPG-KEY-CentOS-Official
  centos_repo="--repofrompath=centos-appstream,https://mirror.stream.centos.org/9-stream/AppStream/${arch}/os/"
  dnf "$centos_repo" module enable -y "postgresql:${postgres_major}"
  dnf "$centos_repo" install -y --downloadonly --downloaddir=/tmp postgresql postgresql-private-libs
  mv /tmp/postgresql-private-libs-*.rpm "${output_dir}/rpms/postgres-libs.rpm"
  mv /tmp/postgresql-*.rpm "${output_dir}/rpms/postgres.rpm"
else
  pg_rhel_major=9
  postgres_repo_url="https://download.postgresql.org/pub/repos/yum/reporpms/EL-${pg_rhel_major}-${arch}/pgdg-redhat-repo-latest.noarch.rpm"
  dnf install --disablerepo='*' -y "${postgres_repo_url}"
  download_dir=$(mktemp -d)
  trap 'rm -rf "$download_dir"' EXIT

  dnf install -y --downloadonly --downloaddir="$download_dir" \
    "postgresql${postgres_major}.${arch}" "postgresql${postgres_major}-libs.${arch}"

  shopt -s nullglob
  postgres_rpms=("$download_dir"/postgresql"${postgres_major}"-[0-9]*."${arch}".rpm)
  postgres_libs_rpms=("$download_dir"/postgresql"${postgres_major}"-libs-[0-9]*."${arch}".rpm)
  if [[ ${#postgres_rpms[@]} -ne 1 || ${#postgres_libs_rpms[@]} -ne 1 ]]; then
    echo "Expected exactly one PostgreSQL client RPM and one library RPM in $download_dir" >&2
    exit 1
  fi
  mv "${postgres_rpms[0]}" "${output_dir}/rpms/postgres.rpm"
  mv "${postgres_libs_rpms[0]}" "${output_dir}/rpms/postgres-libs.rpm"
fi

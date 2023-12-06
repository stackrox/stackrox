#!/usr/bin/env bash

set -euo pipefail

arch=$(uname -m)
goarch="$arch"
dnf_list_args=()
if [[ "$arch" == "x86_64" ]]; then
    goarch="amd64"
elif [[ "$arch" == "aarch64" ]]; then
    goarch="arm64"
    dnf_list_args=('--nogpgcheck')
elif [[ "$arch" == "arm64" ]]; then
    arch="aarch64"
    dnf_list_args=('--nogpgcheck')
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
    echo "WARNING: Architecture ${goarch} is not spported by delve. Rerun with DEBUG_BUILD=no"
  fi
fi

mkdir -p "$output_dir/rpms"
# Install all the required compression packages for RocksDB to compile for amd64. RocksDB is not required for other architectures.
if [[ "$goarch" == "amd64" ]]; then
  rpm_url="http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/snappy-1.1.8-3.el8.x86_64.rpm"
  curl --retry 3 --silent --show-error -f -o "${output_dir}/rpms/snappy.rpm" "${rpm_url}"
fi

if [[ "$arch" == "s390x" ]]; then
  yum install -y --downloadonly --downloaddir=/tmp postgresql postgresql-private-libs
  mv /tmp/postgresql-private-libs-*.rpm "${output_dir}/rpms/postgres-libs.rpm"
  mv /tmp/postgresql-*.rpm "${output_dir}/rpms/postgres.rpm"
else
  postgres_major=13
  pg_rhel_major=8
  postgres_repo_url="https://download.postgresql.org/pub/repos/yum/reporpms/EL-${pg_rhel_major}-${arch}/pgdg-redhat-repo-latest.noarch.rpm"
  dnf install --disablerepo='*' -y "${postgres_repo_url}"
  postgres_minor=$(dnf list ${dnf_list_args[@]+"${dnf_list_args[@]}"} --disablerepo='*' ""--enablerepo=pgdg${postgres_major} -y postgresql${postgres_major}-server."$arch" | tail -n 1 | awk '{print $2}')
  postgres_minor="$postgres_minor.$arch"

  postgres_url="https://download.postgresql.org/pub/repos/yum/${postgres_major}/redhat/rhel-${pg_rhel_major}-${arch}"
  curl --retry 3 -sS --fail -o "${output_dir}/rpms/postgres.rpm" "${postgres_url}/postgresql${postgres_major}-${postgres_minor}.rpm"
  curl --retry 3 -sS --fail -o "${output_dir}/rpms/postgres-libs.rpm" "${postgres_url}/postgresql${postgres_major}-libs-${postgres_minor}.rpm"
fi

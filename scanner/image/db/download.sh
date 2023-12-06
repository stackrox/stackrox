#!/bin/bash

set -euo pipefail

postgres_major=15
pg_rhel_major=8

arch="$(uname -m)"
dnf_list_args=()
if [[ "$arch" == "arm64" ]]; then
  arch="aarch64"
  # Workaround for local Darwin ARM64 builds due to "Error: Failed to download metadata for repo 'pgdg14': repomd.xml GPG signature verification error: Bad GPG signature"
  dnf_list_args=('--nogpgcheck')
fi
output_dir="/rpms"
mkdir $output_dir

if [[ "$arch" == "s390x" ]]; then
  yum install -y --downloadonly --downloaddir=/tmp postgresql postgresql-private-libs postgresql-server postgresql-contrib
  mv /tmp/postgresql-contrib-*.rpm "${output_dir}/postgres-contrib.rpm"
  mv /tmp/postgresql-server-*.rpm "${output_dir}/postgres-server.rpm"
  mv /tmp/postgresql-private-libs-*.rpm "${output_dir}/postgres-libs.rpm"
  mv /tmp/postgresql-*.rpm "${output_dir}/postgres.rpm"
else
  postgres_repo_url="https://download.postgresql.org/pub/repos/yum/reporpms/EL-${pg_rhel_major}-${arch}/pgdg-redhat-repo-latest.noarch.rpm"
  dnf install --disablerepo='*' -y "${postgres_repo_url}"
  postgres_minor=$(dnf list ${dnf_list_args[@]+"${dnf_list_args[@]}"} --disablerepo='*' --enablerepo=pgdg${postgres_major} -y postgresql${postgres_major}-server.$arch | tail -n 1 | awk '{print $2}')
  postgres_minor="$postgres_minor.$arch"

  postgres_url="https://download.postgresql.org/pub/repos/yum/${postgres_major}/redhat/rhel-${pg_rhel_major}-${arch}"
  curl --retry 3 -sS --fail -o "${output_dir}/postgres.rpm" "${postgres_url}/postgresql${postgres_major}-${postgres_minor}.rpm"
  curl --retry 3 -sS --fail -o "${output_dir}/postgres-server.rpm" "${postgres_url}/postgresql${postgres_major}-server-${postgres_minor}.rpm"
  curl --retry 3 -sS --fail -o "${output_dir}/postgres-libs.rpm" "${postgres_url}/postgresql${postgres_major}-libs-${postgres_minor}.rpm"
  curl --retry 3 -sS --fail -o "${output_dir}/postgres-contrib.rpm" "${postgres_url}/postgresql${postgres_major}-contrib-${postgres_minor}.rpm"
fi

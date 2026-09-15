#!/bin/bash

set -euo pipefail

pg_rhel_major=9
arch="$(uname -m)"
dnf_list_args=()
if [[ "$arch" == "arm64" ]]; then
  arch="aarch64"
  # Workaround for local Darwin ARM64 builds due to GPG signature errors.
  dnf_list_args=('--nogpgcheck')
fi
output_dir=/rpms
mkdir -p "$output_dir"

download_pgdg() {
  local postgres_major=$1
  local postgres_repo_url="https://download.postgresql.org/pub/repos/yum/reporpms/EL-${pg_rhel_major}-${arch}/pgdg-redhat-repo-latest.noarch.rpm"
  dnf install --disablerepo='*' -y "${postgres_repo_url}"

  local postgres_minor
  postgres_minor=$(dnf list ${dnf_list_args[@]+"${dnf_list_args[@]}"} \
    --disablerepo='*' --enablerepo="pgdg${postgres_major}" -y \
    "postgresql${postgres_major}-server.${arch}" | tail -n 1 | awk '{print $2}')
  postgres_minor="${postgres_minor}.${arch}"

  local postgres_url="https://download.postgresql.org/pub/repos/yum/${postgres_major}/redhat/rhel-${pg_rhel_major}-${arch}"
  curl --retry 3 -sS --fail -o "${output_dir}/postgresql${postgres_major}.rpm" \
    "${postgres_url}/postgresql${postgres_major}-${postgres_minor}.rpm"
  curl --retry 3 -sS --fail -o "${output_dir}/postgresql${postgres_major}-server.rpm" \
    "${postgres_url}/postgresql${postgres_major}-server-${postgres_minor}.rpm"
  curl --retry 3 -sS --fail -o "${output_dir}/postgresql${postgres_major}-libs.rpm" \
    "${postgres_url}/postgresql${postgres_major}-libs-${postgres_minor}.rpm"
  curl --retry 3 -sS --fail -o "${output_dir}/postgresql${postgres_major}-contrib.rpm" \
    "${postgres_url}/postgresql${postgres_major}-contrib-${postgres_minor}.rpm"
}

download_centos_upgrade() {
  centos_stream_url="https://mirror.stream.centos.org/9-stream/AppStream/${arch}/os/"
  centos_stream_args=(
    --disablerepo='*'
    --repofrompath="centos-stream-appstream,${centos_stream_url}"
    --enablerepo=centos-stream-appstream
    --forcearch="${arch}"
    --setopt=centos-stream-appstream.module_hotfixes=true
  )
  postgres_minor=$(dnf repoquery "${centos_stream_args[@]}" \
    --qf '%{version}-%{release}' postgresql-server | \
    awk '$1 ~ /^15\./' | sort -V | tail -n 1)
  dnf download "${centos_stream_args[@]}" --destdir="${output_dir}" \
    "postgresql-upgrade-${postgres_minor}.${arch}"
  mv "${output_dir}/postgresql-upgrade-${postgres_minor}.${arch}.rpm" "${output_dir}/postgresql-upgrade.rpm"
}

if [[ "$arch" == "s390x" ]]; then
  # PGDG does not publish EL9 s390x packages. CentOS Stream AppStream has
  # PostgreSQL 15 for s390x.
  download_centos_upgrade
  dnf download "${centos_stream_args[@]}" --destdir="${output_dir}" \
    "postgresql-${postgres_minor}.${arch}" \
    "postgresql-server-${postgres_minor}.${arch}" \
    "postgresql-private-libs-${postgres_minor}.${arch}" \
    "postgresql-contrib-${postgres_minor}.${arch}"
  mv "${output_dir}/postgresql-${postgres_minor}.${arch}.rpm" "${output_dir}/postgresql15.rpm"
  mv "${output_dir}/postgresql-server-${postgres_minor}.${arch}.rpm" "${output_dir}/postgresql15-server.rpm"
  mv "${output_dir}/postgresql-private-libs-${postgres_minor}.${arch}.rpm" "${output_dir}/postgresql15-libs.rpm"
  mv "${output_dir}/postgresql-contrib-${postgres_minor}.${arch}.rpm" "${output_dir}/postgresql15-contrib.rpm"
else
  download_pgdg 15
  download_centos_upgrade
fi

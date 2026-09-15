#!/bin/bash

set -euo pipefail

postgres_major=15
pg_rhel_major=9
arch="$(uname -m)"
[[ "$arch" == "arm64" ]] && arch="aarch64"
output_dir=/rpms
mkdir -p "$output_dir"

if [[ "$arch" == "s390x" ]]; then
  # UBI9 does not publish postgresql-server or postgresql-contrib for s390x.
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
    "postgresql-${postgres_minor}.${arch}" \
    "postgresql-server-${postgres_minor}.${arch}" \
    "postgresql-private-libs-${postgres_minor}.${arch}" \
    "postgresql-contrib-${postgres_minor}.${arch}"
  mv "${output_dir}/postgresql-${postgres_minor}.${arch}.rpm" "${output_dir}/postgres.rpm"
  mv "${output_dir}/postgresql-server-${postgres_minor}.${arch}.rpm" "${output_dir}/postgres-server.rpm"
  mv "${output_dir}/postgresql-private-libs-${postgres_minor}.${arch}.rpm" "${output_dir}/postgres-libs.rpm"
  mv "${output_dir}/postgresql-contrib-${postgres_minor}.${arch}.rpm" "${output_dir}/postgres-contrib.rpm"
else
  repo_url="https://download.postgresql.org/pub/repos/yum/reporpms/EL-${pg_rhel_major}-${arch}/pgdg-redhat-repo-latest.noarch.rpm"
  dnf install --disablerepo='*' -y "$repo_url"
  postgres_minor="$(dnf list --disablerepo='*' --enablerepo="pgdg${postgres_major}" -y "postgresql${postgres_major}-server.${arch}" | tail -n 1 | awk '{print $2}')"
  postgres_minor="${postgres_minor}.${arch}"
  postgres_url="https://download.postgresql.org/pub/repos/yum/${postgres_major}/redhat/rhel-${pg_rhel_major}-${arch}"
  curl --retry 3 -sS --fail -o "$output_dir/postgres.rpm" "$postgres_url/postgresql${postgres_major}-${postgres_minor}.rpm"
  curl --retry 3 -sS --fail -o "$output_dir/postgres-server.rpm" "$postgres_url/postgresql${postgres_major}-server-${postgres_minor}.rpm"
  curl --retry 3 -sS --fail -o "$output_dir/postgres-libs.rpm" "$postgres_url/postgresql${postgres_major}-libs-${postgres_minor}.rpm"
  curl --retry 3 -sS --fail -o "$output_dir/postgres-contrib.rpm" "$postgres_url/postgresql${postgres_major}-contrib-${postgres_minor}.rpm"
fi

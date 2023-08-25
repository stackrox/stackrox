#! /bin/bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

if [[ -z $1 ]]; then
  echo "db zip file must be passed"
  exit 1
fi

rm -rf "${DIR}/database-restore"
mkdir -p "${DIR}/database-restore"
unzip -d "${DIR}/database-restore/expanded" "$1"
mkdir -p "${DIR}/database-restore/expanded/rocksdb"
tar -xvf "${DIR}/database-restore/expanded/rocks.db" -C "${DIR}/database-restore/expanded/rocksdb"
mkdir -p "${DIR}/database-restore/full/rocksdb"
go run "${DIR}/expand/main.go" --backup  "${DIR}/database-restore/expanded/rocksdb" --restored "${DIR}/database-restore/full/rocksdb"
mv "${DIR}/database-restore/expanded/bolt.db" "${DIR}/database-restore/full/bolt.db"
rm -rf "${DIR}/database-restore/expanded"

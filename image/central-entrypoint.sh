#!/bin/sh

set -e

# When running as the root user, chown the directories
# and then exec as the non-root user.
# This is important for upgrades; for fresh installs,
# the chown has already taken effect in the Dockerfile.
if [ "$(id -u)" == 0 ]; then
     err=0
     [ ! -d /var/lib/stackrox ] || chown -R 4000:4000 /var/lib/stackrox || err=1
     [ ! -d /var/log/stackrox ] || chown -R 4000:4000 /var/log/stackrox || err=1
     [ ! -d /etc/ssl ] || chown -R 4000:4000 /etc/ssl || err=1
     [ ! -d /etc/pki/ca-trust ] || chown -R 4000:4000 /etc/pki/ca-trust || err=1
     chown -R 4000:4000 /tmp || err=1
     chown -R 4000:4000 /stackrox/data || err=1

     if [ $err -ne 0 ]; then
        echo >&2 "Warning: failed to change permissions of one or more directories. Startup may fail."
     fi

     exec su-exec 4000:4000 "$0" "$@"
fi

restore-all-dir-contents
import-additional-cas

dbpath="/var/lib/stackrox/stackrox.db"
backup_dbpath="/var/lib/stackrox/stackrox.db.pre-rocksdb-snapshot"

# If we want to move to RocksDB, we want to backup bolt in it's current state
if [ "${ROX_ROCKSDB}" == "true" ] && [ -f "$dbpath" ] && [ ! -f "$backup_dbpath" ]; then
    # backup bolt db before cutting over BadgerDB to RocksDB
  echo >&2 "Backing up BoltDB before migration to RocksDB"
  echo >&2 "Copying ${dbpath} to ${backup_dbpath}.tmp"
  cp "$dbpath" "${backup_dbpath}.tmp"
  echo >&2 "Atomically renaming ${backup_dbpath}.tmp to ${backup_dbpath}"
  mv "${backup_dbpath}.tmp" "${backup_dbpath}"
  echo >&2 "Successfully backed up BoltDB"

  # Remove currently existing RocksDB before migration from BadgerDB to RocksDB
  # This covers the case, where someone has flipped the flag repeatedly
  if [ -d /var/lib/stackrox/rocksdb ]; then
    rocksdb_backup="/var/lib/stackrox/rocksdb-$(date +"%Y%m%d_%H%M%S")"
    echo >&2 "RocksDB directory already exists. Saving /var/lib/stackrox/rocksdb to ${rocksdb_backup}"
    mv "/var/lib/stackrox/rocksdb" "${rocksdb_backup}"
  fi
fi

# If we don't want RocksDB, then copy the backup if it exists back to the original location and we should
# be able to start correctly with a RocksDB and BadgerDB
if [ "${ROX_ROCKSDB}" != "true" ] && [ -f "$backup_dbpath" ]; then
  echo  >&2 "Reverting back from RocksDB to saved snapshot of BoltDB and BadgerDB"
  # Restore from backup taken
  echo >&2 "Copying from ${backup_dbpath} to ${dbpath}"
  mv "$backup_dbpath" "$dbpath"
  echo >&2 "Removing indexes to force a rebuild"
  rm -rf /var/lib/stackrox/scorch.bleve /var/lib/stackrox/index
fi

exec /stackrox/start-central.sh "$@"

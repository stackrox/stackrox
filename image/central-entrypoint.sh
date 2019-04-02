#!/bin/sh

set -e

# When running as the root user, chown the directories
# and then exec as the non-root user.
# This is important for upgrades; for fresh installs,
# the chown has already taken effect in the Dockerfile.
if [ "$(id -u)" == 0 ]; then
     [ ! -d /var/lib/stackrox ] || chown -R 4000 /var/lib/stackrox
     [ ! -d /var/log/stackrox ] || chown -R 4000 /var/log/stackrox
     chown -R 4000 /tmp
     chown -R 4000 /etc/ssl
     exec su-exec 4000 "$0" "$@"
fi

if [ -d /usr/local/share/ca-certificates -a "$(find /usr/local/share/ca-certificates -name '*.crt' -maxdepth 1 | wc -l)" -gt 0 ]; then
  if [ -f /etc/redhat-release ]; then
    # On RHEL
    cp -L /usr/local/share/ca-certificates/* /etc/pki/ca-trust/source/anchors
    update-ca-trust
  else
    # On Alpine
    update-ca-certificates
  fi
fi

[ ! -d /var/lib/stackrox/badgerdb ] || chmod u+x /var/lib/stackrox/badgerdb

restore-central-db
migrator
exec central "$@"

#!/bin/sh

set -e

# When running as the root user, chown the directories
# and then exec as the non-root user.
# This is important for upgrades; for fresh installs,
# the chown has already taken effect in the Dockerfile.
if [ "$(id -u)" == 0 ]; then
     [ ! -d /var/lib/stackrox ] || chown -R 4000 /var/lib/stackrox
     [ ! -d /var/log/stackrox ] || chown -R 4000 /var/log/stackrox
     [ ! -d /etc/ssl ] || chown -R 4000 /etc/ssl
     [ ! -d /etc/pki/ca-trust ] || chown -R 4000 /etc/pki/ca-trust
     chown -R 4000 /tmp
     chown -R 4000 /stackrox/data
     exec su-exec 4000 "$0" "$@"
fi

restore-all-dir-contents
import-additional-cas

[ ! -d /var/lib/stackrox/badgerdb ] || chmod u+x /var/lib/stackrox/badgerdb

exec /stackrox/start-central.sh "$@"

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

     if [ $err -ne 0 ]; then
        echo >&2 "Warning: failed to change permissions of one or more directories. Startup may fail."
     fi

     exec su-exec 4000:4000 "$0" "$@"
fi

restore-all-dir-contents
import-additional-cas

exec /stackrox/start-central.sh "$@"

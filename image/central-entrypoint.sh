#!/bin/sh

set -e

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

restore-central-db
migrator
exec central "$@"

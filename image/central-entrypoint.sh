#!/bin/sh

set -e

update-ca-certificates

restore-central-db
migrator
exec central "$@"

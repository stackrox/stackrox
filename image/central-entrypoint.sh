#!/bin/sh

set -e

update-ca-certificates

migrator
exec central "$@"

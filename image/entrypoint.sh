#!/bin/sh

set -e

update-ca-certificates

exec "$@"

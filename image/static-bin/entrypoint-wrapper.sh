#!/bin/sh

set -e

restore-all-dir-contents
import-additional-cas

exec "/stackrox/bin/$(basename "$0")" "$@"

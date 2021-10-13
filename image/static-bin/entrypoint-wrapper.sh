#!/bin/sh

set -euo pipefail

restore-all-dir-contents
import-additional-cas

exec "/stackrox/bin/$(basename "$0")" "$@"

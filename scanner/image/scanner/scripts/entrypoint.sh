#!/usr/bin/env bash

set -euo pipefail

fix-etc-pki-permissions
restore-all-dir-contents
import-additional-cas

exec scanner "$@"

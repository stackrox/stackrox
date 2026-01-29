#!/usr/bin/env bash

set -euo pipefail

restore-all-dir-contents
bundle-ca-trust

exec scanner "$@"

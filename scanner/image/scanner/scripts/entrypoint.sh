#!/usr/bin/env bash

set -euo pipefail

restore-all-dir-contents
import-additional-cas

exec scanner "$@"

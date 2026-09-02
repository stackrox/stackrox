#!/usr/bin/env bash
set -euo pipefail

# Scanner V2 is not installed. Callers should skip V2-only assertions rather than
# waiting for deploy/scanner.

echo "Scanner V2 is not deployed" >&2
exit 1

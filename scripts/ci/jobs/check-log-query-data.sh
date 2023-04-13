#!/usr/bin/env bash

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../scripts/lib.sh
source "$SCRIPTS_ROOT/scripts/lib.sh"

set -euo pipefail

echo 'Ensure that query.Data is never logged directly in release builds as it could potentially leak sensitive data'
echo "Fix by using redactedQueryData(query) instead. This will ensure it only gets logged in debug builds"

"$SCRIPTS_ROOT/scripts/check-log-query-data.sh"

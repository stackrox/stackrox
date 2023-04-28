#!/usr/bin/env bash

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../../.. && pwd)"
# shellcheck source=../../../scripts/lib.sh
source "$SCRIPTS_ROOT/scripts/lib.sh"

set -euo pipefail

echo 'Ensure that there are no TODO references that the developer has marked as blocking a merge'
echo "Matches comments of the form TODO(x), where x can be \"DO NOT MERGE/don't-merge\"/\"dont-merge\"/similar"

"$SCRIPTS_ROOT/scripts/check-todos.sh" 'do\s?n.*merge'

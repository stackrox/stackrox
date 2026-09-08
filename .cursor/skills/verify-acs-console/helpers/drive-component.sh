#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PLATFORM="${ROOT}/ui/apps/platform"
FEATURE="${1:-}"

declare -A FEATURE_SPECS=(
    [dashboard]="src/Containers/Dashboard/Widgets/ViolationsByPolicySeverity.cy.jsx"
    [violations]="src/Containers/Dashboard/Widgets/ViolationsByPolicySeverity.cy.jsx"
    [code-viewer]="src/Components/CodeViewer.cy.jsx"
    [scope-bar]="src/Containers/Dashboard/ScopeBar.cy.jsx"
)

if [[ -z "${FEATURE}" ]]; then
    echo "usage: $0 <feature-key>"
    echo "keys: ${!FEATURE_SPECS[*]}"
    exit 1
fi

SPEC="${FEATURE_SPECS[${FEATURE}]:-}"
if [[ -z "${SPEC}" ]]; then
    echo "unknown feature key: ${FEATURE}"
    exit 1
fi

cd "${PLATFORM}"
TZ=UTC npm run test-component -- --spec "${SPEC}"

#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PLATFORM="${ROOT}/ui/apps/platform"
RUN_ID="${1:-$(date -u +%Y%m%dT%H%M%SZ)}"
DEST="/opt/cursor/artifacts/verify-acs-console/${RUN_ID}"
SRC="${PLATFORM}/cypress/test-results/artifacts"

mkdir -p "${DEST}"

if [[ -d "${SRC}" ]]; then
    mkdir -p "${DEST}"
    cp -r "${SRC}/." "${DEST}/"
    echo "evidence: copied ${SRC} -> ${DEST}"
else
    echo "evidence: no cypress artifacts at ${SRC}"
fi

# Persist a machine-readable manifest outside cypress cleanup paths.
MANIFEST="${ROOT}/.cursor/skills/verify-acs-console/.last-evidence-path"
echo "${DEST}" >"${MANIFEST}"
echo "evidence: manifest ${MANIFEST}"
ls -la "${DEST}" 2>/dev/null || true

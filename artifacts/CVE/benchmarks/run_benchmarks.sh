#!/bin/bash
# Run CVE prototype benchmark queries against a local Postgres instance.
# Usage: ./run_benchmarks.sh [output_file]
#
# Environment variables:
#   DB_HOST  - Postgres host (default: localhost)
#   DB_PORT  - Postgres port (default: 5432)
#   DB_USER  - Postgres user (default: $USER)
#   DB_NAME  - Database name (default: central_active)

set -euo pipefail

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-$USER}"
DB_NAME="${DB_NAME:-central_active}"
OUTPUT="${1:-results_$(date +%Y%m%d_%H%M%S).txt}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
QUERIES_FILE="$SCRIPT_DIR/queries.sql"

if [[ ! -f "$QUERIES_FILE" ]]; then
    echo "ERROR: queries.sql not found at $QUERIES_FILE" >&2
    exit 1
fi

echo "=== CVE Prototype Benchmark ===" | tee "$OUTPUT"
echo "Date: $(date)" | tee -a "$OUTPUT"
echo "Database: ${DB_NAME}@${DB_HOST}:${DB_PORT}" | tee -a "$OUTPUT"
echo "" | tee -a "$OUTPUT"

psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
    -f "$QUERIES_FILE" 2>&1 | tee -a "$OUTPUT"

echo "" | tee -a "$OUTPUT"
echo "=== Complete ===" | tee -a "$OUTPUT"
echo "Results saved to: $OUTPUT"

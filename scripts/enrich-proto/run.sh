#!/usr/bin/env bash
# run.sh — Proto enrichment runner for a single ACS service.
#
# Usage:
#   ./scripts/enrich-proto/run.sh <task-id> <service-name>
#   ./scripts/enrich-proto/run.sh <task-id> <service-name> --run
#
# With --run: executes the enrichment prompt directly via the `claude` CLI.
# Without --run: prints the subagent prompt to stdout.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
WORKFLOW_TEMPLATE="${SCRIPT_DIR}/WORKFLOW.md"

# ---------------------------------------------------------------------------
# Service mapping: SERVICE_NAME -> "proto_file|impl_dir|storage_proto"
# "none" means no storage proto applies.
# ---------------------------------------------------------------------------
declare -A SERVICE_MAP
SERVICE_MAP["AlertService"]="proto/api/v1/alert_service.proto|central/alert/service/|proto/storage/alert.proto"
SERVICE_MAP["DeploymentService"]="proto/api/v1/deployment_service.proto|central/deployment/service/|proto/storage/deployment.proto"
SERVICE_MAP["ImageService"]="proto/api/v1/image_service.proto|central/image/service/|proto/storage/image.proto"
SERVICE_MAP["PolicyService"]="proto/api/v1/policy_service.proto|central/policy/service/|proto/storage/policy.proto"
SERVICE_MAP["ClusterService"]="proto/api/v1/cluster_service.proto|central/cluster/service/|proto/storage/cluster.proto"
SERVICE_MAP["CVEService"]="proto/api/v1/cve_service.proto|central/cve/service/|none"
SERVICE_MAP["SearchService"]="proto/api/v1/search_service.proto|central/search/service/|none"
SERVICE_MAP["ComplianceService"]="proto/api/v1/compliance_service.proto|central/compliance/service/|proto/storage/compliance.proto"
SERVICE_MAP["ConfigService"]="proto/api/v1/config_service.proto|central/config/service/|proto/storage/config.proto"
SERVICE_MAP["ReportService"]="proto/api/v1/report_service.proto|central/reports/service/|proto/storage/report_configuration.proto"
SERVICE_MAP["VulnMgmtService"]="proto/api/v1/vuln_mgmt_service.proto|central/vulnmgmt/service/|none"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

usage() {
  echo "Usage: $0 <task-id> <service-name> [--run]"
  echo ""
  echo "Available services:"
  for svc in $(echo "${!SERVICE_MAP[@]}" | tr ' ' '\n' | sort); do
    echo "  ${svc}"
  done
  exit 1
}

die() {
  echo "ERROR: $*" >&2
  exit 1
}

# Derive the swagger output path from the proto file path.
# e.g. proto/api/v1/alert_service.proto -> generated/api/v1/alert_service.swagger.json
swagger_path_for() {
  local proto_file="$1"
  # Strip leading proto/ and replace extension with .swagger.json under generated/
  echo "${proto_file/proto\//generated/}" | sed 's/\.proto$/.swagger.json/'
}

# ---------------------------------------------------------------------------
# Parse arguments
# ---------------------------------------------------------------------------

if [[ $# -lt 2 ]]; then
  usage
fi

TASK_ID="$1"
SERVICE_NAME="$2"
RUN_MODE=false

for arg in "${@:3}"; do
  case "${arg}" in
    --run) RUN_MODE=true ;;
    *) die "Unknown argument: ${arg}" ;;
  esac
done

# ---------------------------------------------------------------------------
# Look up the service
# ---------------------------------------------------------------------------

if [[ -z "${SERVICE_MAP[${SERVICE_NAME}]+_}" ]]; then
  die "Unknown service '${SERVICE_NAME}'. Run without arguments to see available services."
fi

IFS='|' read -r PROTO_FILE IMPL_DIR STORAGE_FILE <<< "${SERVICE_MAP[${SERVICE_NAME}]}"
SWAGGER_FILE="$(swagger_path_for "${PROTO_FILE}")"

# For the prompt we want the full path to the impl dir, but keep the template
# placeholder as the directory — the subagent will find the actual .go file.
IMPL_FILE="${IMPL_DIR}"

# ---------------------------------------------------------------------------
# Fill in the WORKFLOW.md template
# ---------------------------------------------------------------------------

if [[ ! -f "${WORKFLOW_TEMPLATE}" ]]; then
  die "WORKFLOW.md template not found at ${WORKFLOW_TEMPLATE}"
fi

PROMPT="$(sed \
  -e "s|{{SERVICE_NAME}}|${SERVICE_NAME}|g" \
  -e "s|{{PROTO_FILE}}|${PROTO_FILE}|g" \
  -e "s|{{IMPL_FILE}}|${IMPL_FILE}|g" \
  -e "s|{{STORAGE_FILE}}|${STORAGE_FILE}|g" \
  -e "s|{{SWAGGER_FILE}}|${SWAGGER_FILE}|g" \
  -e "s|{{TASK_ID}}|${TASK_ID}|g" \
  "${WORKFLOW_TEMPLATE}")"

# ---------------------------------------------------------------------------
# Output or execute
# ---------------------------------------------------------------------------

if [[ "${RUN_MODE}" == false ]]; then
  echo "${PROMPT}"
  exit 0
fi

# --run mode: execute via claude CLI
if ! command -v claude &>/dev/null; then
  die "'claude' CLI not found in PATH. Install it or omit --run to print the prompt."
fi

echo "==> Running enrichment for ${SERVICE_NAME} (task ${TASK_ID}) via claude CLI..." >&2
echo "${PROMPT}" | claude --print -

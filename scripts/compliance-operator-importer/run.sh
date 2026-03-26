#!/usr/bin/env bash
# run.sh — Run the compliance-operator-importer via container.
#
# Automatically mounts kubeconfig files and forwards ACS auth env vars
# so you don't have to spell out docker/podman flags manually.
#
# USAGE:
#   ./run.sh --endpoint central.example.com --dry-run
#   ./run.sh --endpoint central.example.com --context my-cluster
#
# ENVIRONMENT (read from host, forwarded to container):
#   KUBECONFIG            Colon-separated kubeconfig paths (default: ~/.kube/config)
#   ROX_ENDPOINT          ACS Central URL (alternative to --endpoint)
#   ROX_API_TOKEN         API token auth
#   ROX_ADMIN_PASSWORD    Basic auth password
#   ROX_ADMIN_USER        Basic auth username (default: admin)
#
# IMAGE override:
#   IMAGE=my-registry/co-importer:v1 ./run.sh --endpoint ...

set -euo pipefail

IMAGE="${IMAGE:-localhost/compliance-operator-importer:latest}"
CONTAINER_RT="${CONTAINER_RT:-$(command -v podman 2>/dev/null || echo docker)}"

# ── Kubeconfig mounts ────────────────────────────────────────────────────────

kubeconfig_paths="${KUBECONFIG:-$HOME/.kube/config}"

mount_args=()
container_paths=()
i=0

IFS=':' read -ra kc_files <<< "$kubeconfig_paths"
for f in "${kc_files[@]}"; do
    f="${f/#\~/$HOME}"
    if [[ ! -f "$f" ]]; then
        echo "WARNING: kubeconfig not found, skipping: $f" >&2
        continue
    fi
    target="/kubeconfig/config-${i}"
    mount_args+=(-v "$f:$target:ro")
    container_paths+=("$target")
    ((++i))
done

if [[ ${#container_paths[@]} -eq 0 ]]; then
    echo "ERROR: no kubeconfig files found" >&2
    exit 1
fi

# Join container paths with ':' for the in-container KUBECONFIG.
joined=$(IFS=':'; echo "${container_paths[*]}")

# ── Auth env vars ────────────────────────────────────────────────────────────

env_args=(-e "KUBECONFIG=$joined")

for var in ROX_ENDPOINT ROX_API_TOKEN ROX_ADMIN_PASSWORD ROX_ADMIN_USER; do
    if [[ -n "${!var:-}" ]]; then
        env_args+=(-e "$var=${!var}")
    fi
done

# ── Run ──────────────────────────────────────────────────────────────────────

exec "$CONTAINER_RT" run --rm \
    "${mount_args[@]}" \
    "${env_args[@]}" \
    "$IMAGE" \
    "$@"

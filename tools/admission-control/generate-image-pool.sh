#!/usr/bin/env bash
# generate-image-pool.sh -- Build a list of unique quay.io image refs.
#
# Usage:
#   ./generate-image-pool.sh [NUM_IMAGES]
#
# Outputs one image ref per line to stdout. Uses the quay.io tag API to
# enumerate tags across popular repositories. Falls back to a hardcoded
# pool of 20 images when the API is unreachable.
#
# Results are cached in /tmp/image-pool-cache.txt to avoid repeated API calls.
set -euo pipefail

TARGET="${1:-20}"
CACHE_FILE="/tmp/image-pool-cache.txt"
CACHE_MAX_AGE_S=3600  # 1 hour

FALLBACK_POOL=(
    "quay.io/centos/centos:7"
    "quay.io/centos/centos:stream9"
    "quay.io/fedora/fedora:37"
    "quay.io/fedora/fedora:38"
    "quay.io/fedora/fedora:39"
    "quay.io/fedora/fedora:40"
    "quay.io/almalinux/almalinux:8"
    "quay.io/almalinux/almalinux:9"
    "quay.io/rockylinux/rockylinux:8"
    "quay.io/rockylinux/rockylinux:9"
    "quay.io/prometheus/prometheus:v2.48.0"
    "quay.io/prometheus/prometheus:v2.45.0"
    "quay.io/prometheus/alertmanager:v0.26.0"
    "quay.io/prometheus/node-exporter:v1.7.0"
    "quay.io/prometheus/blackbox-exporter:v0.24.0"
    "quay.io/prometheus/pushgateway:v1.6.2"
    "quay.io/coreos/etcd:v3.5.10"
    "quay.io/coreos/etcd:v3.5.9"
    "quay.io/strimzi/kafka:0.38.0-kafka-3.6.0"
    "quay.io/keycloak/keycloak:23.0"
)

REPOS=(
    centos/centos
    fedora/fedora
    almalinux/almalinux
    rockylinux/rockylinux
    prometheus/prometheus
    prometheus/alertmanager
    prometheus/node-exporter
    prometheus/blackbox-exporter
    prometheus/pushgateway
    coreos/etcd
    strimzi/kafka
    keycloak/keycloak
    jetstack/cert-manager-controller
    jetstack/cert-manager-webhook
    jetstack/cert-manager-cainjector
    argoproj/argocd
    minio/minio
    minio/mc
    cilium/cilium
    cilium/operator-generic
    cilium/hubble-relay
    metallb/speaker
    metallb/controller
    grafana/grafana
    grafana/loki
    grafana/promtail
    thanos-io/thanos
    bitnami/redis
    bitnami/nginx
    bitnami/postgresql
    ceph/ceph
    quay/quay
    quay/clair
    calico/node
    calico/cni
    calico/kube-controllers
    brancz/kube-rbac-proxy
    coreos/flannel
    oauth2-proxy/oauth2-proxy
)

use_fallback() {
    printf '%s\n' "${FALLBACK_POOL[@]}" | head -n "$TARGET"
}

cache_is_fresh() {
    [[ -f "$CACHE_FILE" ]] || return 1
    local cache_lines
    cache_lines=$(wc -l < "$CACHE_FILE")
    [[ "$cache_lines" -ge "$TARGET" ]] || return 1
    if [[ "$(uname)" == "Darwin" ]]; then
        local mod_epoch
        mod_epoch=$(stat -f '%m' "$CACHE_FILE")
        local now_epoch
        now_epoch=$(date +%s)
        (( (now_epoch - mod_epoch) < CACHE_MAX_AGE_S ))
    else
        find "$CACHE_FILE" -maxdepth 0 -mmin "-$(( CACHE_MAX_AGE_S / 60 ))" | grep -q .
    fi
}

if [[ "$TARGET" -le 20 ]]; then
    use_fallback
    exit 0
fi

if cache_is_fresh; then
    head -n "$TARGET" "$CACHE_FILE"
    exit 0
fi

command -v curl &>/dev/null || { echo "ERROR: curl required" >&2; exit 1; }
command -v jq &>/dev/null || { echo "ERROR: jq required for pool generation > 20 images" >&2; exit 1; }

TMPFILE=$(mktemp)
trap 'rm -f "$TMPFILE"' EXIT

for repo in "${REPOS[@]}"; do
    echo "Fetching tags for quay.io/${repo}..." >&2
    tags=$(curl -sf --connect-timeout 5 --max-time 10 \
        "https://quay.io/api/v1/repository/${repo}/tag/?limit=100&onlyActiveTags=true" \
        | jq -r '.tags[].name // empty' 2>/dev/null || true)

    if [[ -z "$tags" ]]; then
        echo "  WARN: no tags returned for ${repo}, skipping" >&2
        continue
    fi

    while IFS= read -r tag; do
        [[ -z "$tag" ]] && continue
        echo "quay.io/${repo}:${tag}" >> "$TMPFILE"
    done <<< "$tags"

    count=$(wc -l < "$TMPFILE")
    if [[ "$count" -ge "$TARGET" ]]; then
        break
    fi
done

count=$(wc -l < "$TMPFILE")
if [[ "$count" -lt "$TARGET" ]]; then
    echo "WARN: only collected ${count} images from API (requested ${TARGET}). Using what we have." >&2
fi

cp "$TMPFILE" "$CACHE_FILE"
head -n "$TARGET" "$CACHE_FILE"

#!/usr/bin/env sh

# Script for migrating to new Helm-style deployment.
# After running this script the state of all
# StackRox K8s resources should be ready for deploying
# using the new Helm chart using 'helm install'.

set -eu

# You can use this script for applying the kubectl commands to the relevant resources directly
# or let it output the necessary kubectl commands for patching the resources to stdout using:
#
#  DRY_RUN=true ./prepare-resource-metadata-for-helm-migration.sh
#
# Further configuration options:
#
#   * The namespace can be configured using the environment variable NAMESPACE
#     (note that it defaults to "stackrox" and that is the only supported namespace).
#
#   * By default this script uses kubectl to verify the existence of the Kubernetes resources before
#     patching them. This can be disabled by setting SKIP_EXISTENCE_CHECK=true.

KUBECTL="${KUBECTL:-kubectl}"
DRY_RUN="${DRY_RUN:-false}"
NAMESPACE="${STACKROX_NAMESPACE:-stackrox}"
SKIP_EXISTENCE_CHECK="${SKIP_EXISTENCE_CHECK:-false}"

die() {
    log "$@"
    exit 1
}

log() {
    echo "$@" >&2
}

if [ "$DRY_RUN" != "false" -a "$DRY_RUN" != "true" ]; then
    die "Unsupported value for DRY_RUN: '$DRY_RUN'"
fi

if [ "$SKIP_EXISTENCE_CHECK" != "false" -a "$SKIP_EXISTENCE_CHECK" != "true" ]; then
    die "Unsupported value for SKIP_EXISTENCE_CHECK: '$SKIP_EXISTENCE_CHECK'"
fi

add_label() {
    if [ "$DRY_RUN" == "true" ]; then
        echo $KUBECTL -n $NAMESPACE label "$kind" "$res" --overwrite "$1=$2"
    else
        $KUBECTL -n $NAMESPACE label "$kind" "$res" --overwrite "$1=$2"
    fi
    log "  Set label $1=$2"
}

add_annotation() {
    if [ "$DRY_RUN" == "true" ]; then
        echo $KUBECTL -n $NAMESPACE annotate "$kind" "$res" --overwrite "$1=$2"
    else
        $KUBECTL -n $NAMESPACE annotate "$kind" "$res" --overwrite "$1=$2"
    fi
    log "  Set annotation $1=$2"
}

patch_resource() {
    kind="$1"
    res="$2"

    if [ "$SKIP_EXISTENCE_CHECK" == "false" ]; then
        $KUBECTL -n $NAMESPACE get "$kind" "$res" >/dev/null 2>&1 || {
            log "Skipping ${kind}/${res}: Resource not known in cluster."
            log
            return
        }
    fi

    log "** Patching resource $kind/$res **"
    add_label "app.kubernetes.io/name" "stackrox"
    add_label "app.kubernetes.io/managed-by" "Helm"
    add_annotation "meta.helm.sh/release-name" "stackrox-central-services"
    add_annotation "meta.helm.sh/release-namespace" "$NAMESPACE"
    log
}

patch_resource "Application" "stackrox"
patch_resource "ClusterRole" "stackrox-central-psp"
patch_resource "ClusterRole" "stackrox-scanner-psp"
patch_resource "ConfigMap" "central-config"
patch_resource "ConfigMap" "central-endpoints"
patch_resource "ConfigMap" "scanner-config"
patch_resource "Deployment" "central"
patch_resource "Deployment" "scanner"
patch_resource "Deployment" "scanner-db"
patch_resource "DestinationRule" "central-internal-no-istio-mtls"
patch_resource "DestinationRule" "scanner-db-internal-no-istio-mtls"
patch_resource "DestinationRule" "scanner-internal-no-istio-mtls"
patch_resource "HorizontalPodAutoscaler" "scanner"
patch_resource "NetworkPolicy" "allow-ext-to-central"
patch_resource "NetworkPolicy" "scanner"
patch_resource "NetworkPolicy" "scanner-db"
patch_resource "PersistentVolumeClaim" "stackrox-db"
patch_resource "PodSecurityPolicy" "stackrox-central"
patch_resource "PodSecurityPolicy" "stackrox-scanner"
patch_resource "Role" "stackrox-central-diagnostics"
patch_resource "RoleBinding" "stackrox-central-diagnostics"
patch_resource "RoleBinding" "stackrox-central-psp"
patch_resource "RoleBinding" "stackrox-scanner-psp"
patch_resource "Route" "central"
patch_resource "Route" "central-mtls"
patch_resource "Secret" "central-default-tls-cert"
patch_resource "Secret" "central-htpasswd"
patch_resource "Secret" "central-license"
patch_resource "Secret" "central-tls"
patch_resource "Secret" "proxy-config"
patch_resource "Secret" "scanner-db-password"
patch_resource "Secret" "scanner-db-tls"
patch_resource "Secret" "scanner-tls"
patch_resource "Secret" "stackrox"
patch_resource "SecurityContextConstraints" "central"
patch_resource "SecurityContextConstraints" "scanner"
patch_resource "Service" "central"
patch_resource "Service" "central-loadbalancer"
patch_resource "Service" "scanner"
patch_resource "Service" "scanner-db"
patch_resource "ServiceAccount" "central"
patch_resource "ServiceAccount" "scanner"

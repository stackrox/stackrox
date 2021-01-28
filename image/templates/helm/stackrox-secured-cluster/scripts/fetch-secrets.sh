#!/bin/sh

# fetch-secrets.sh
# Retrieves StackRox TLS secrets currently stored in the current Kubernetes context, and stores them in a format
# suitable for consumption by the Helm chart.
#
# The YAML bundle is printed to stdout, use output redirection (>filename) to store the output to a file.
# This script supports the following environment variables:
# - KUBECTL: the command to use for kubectl. Spaces will be tokenized by the shell interpreter (default: "kubectl").
# - ROX_NAMESPACE: the namespace in which the current StackRox deployment runs (default: "stackrox")
# - FETCH_CA_ONLY: if set to "true", will create a bundle containing only the CA certificate (default: "false")

DIR="$(cd "$(dirname "$0")" && pwd)"

KUBECTL="${KUBECTL:-kubectl}"
ROX_NAMESPACE="${ROX_NAMESPACE:-stackrox}"

FETCH_CA_ONLY="${FETCH_CA_ONLY:-false}"

case "$FETCH_CA_ONLY" in
  false|0)
    TEMPLATE_FILE="fetched-secrets-bundle.yaml.tpl"
    DESCRIPTION="certificates and keys"
    ;;
  true|1)
    TEMPLATE_FILE="fetched-secrets-bundle-ca-only.yaml.tpl"
    DESCRIPTION="CA certificate only"
    ;;
  *)
    echo >&2 "Invalid value '$FETCH_CA_ONLY' for FETCH_CA_ONLY, only false and true are allowed"
    exit 1
esac

# The leading '#' signs aren't required as they don't go to stdout, but when printing to the console,
# it looks more natural to include them.
echo >&2 "# Fetching $DESCRIPTION from current Kubernetes context (namespace $ROX_NAMESPACE), store"
echo >&2 "# the output in a file and pass it to helm via the -f parameter."

$KUBECTL get --ignore-not-found -n "$ROX_NAMESPACE" \
  secret/sensor-tls secret/collector-tls secret/admission-control-tls \
  -o go-template-file="${DIR}/${TEMPLATE_FILE}" \

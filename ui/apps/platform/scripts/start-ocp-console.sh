#!/usr/bin/env bash

set -euo pipefail

CONSOLE_IMAGE=${CONSOLE_IMAGE:="quay.io/openshift/origin-console:latest"}
CONSOLE_PORT=${CONSOLE_PORT:=9000}
CONSOLE_IMAGE_PLATFORM=${CONSOLE_IMAGE_PLATFORM:="linux/amd64"}

# The ACS backend service URL that will receive the console's proxied requests.
ACS_API_SERVICE_URL=${ACS_API_SERVICE_URL:=https://$(oc -n stackrox get route central -o jsonpath='{.spec.host}')}

# Whether to inject the OCP auth token into the ACS backend service.
ACS_INJECT_OCP_AUTH_TOKEN=${ACS_INJECT_OCP_AUTH_TOKEN:=true}

# Plugin metadata is declared in package.json
PLUGIN_NAME="advanced-cluster-security"

echo "Starting local OpenShift console using ACS API Service URL: $ACS_API_SERVICE_URL"
echo "Injecting OCP auth token: $ACS_INJECT_OCP_AUTH_TOKEN"

export BRIDGE_USER_AUTH="disabled"
export BRIDGE_K8S_MODE="off-cluster"
export BRIDGE_K8S_AUTH="bearer-token"
export BRIDGE_K8S_MODE_OFF_CLUSTER_SKIP_VERIFY_TLS=true
export BRIDGE_K8S_MODE_OFF_CLUSTER_ENDPOINT=$(oc whoami --show-server)
# The monitoring operator is not always installed (e.g. for local OpenShift). Tolerate missing config maps.
set +e
export BRIDGE_K8S_MODE_OFF_CLUSTER_THANOS=$(oc -n openshift-config-managed get configmap monitoring-shared-config -o jsonpath='{.data.thanosPublicURL}' 2>/dev/null)
export BRIDGE_K8S_MODE_OFF_CLUSTER_ALERTMANAGER=$(oc -n openshift-config-managed get configmap monitoring-shared-config -o jsonpath='{.data.alertmanagerPublicURL}' 2>/dev/null)
set -e
export BRIDGE_K8S_AUTH_BEARER_TOKEN=$(oc whoami --show-token 2>/dev/null)
export BRIDGE_USER_SETTINGS_LOCATION="localstorage"
export BRIDGE_I18N_NAMESPACES="plugin__${PLUGIN_NAME}"

# Don't fail if the cluster doesn't have gitops.
set +e
export GITOPS_HOSTNAME=$(oc -n openshift-gitops get route cluster -o jsonpath='{.spec.host}' 2>/dev/null)
set -e
if [ -n "$GITOPS_HOSTNAME" ]; then
    export BRIDGE_K8S_MODE_OFF_CLUSTER_GITOPS="https://$GITOPS_HOSTNAME"
fi

export BRIDGE_PLUGIN_PROXY='{"services":[{"consoleAPIPath":"/api/proxy/plugin/advanced-cluster-security/api-service/","endpoint":"'$ACS_API_SERVICE_URL'", "authorize":'$ACS_INJECT_OCP_AUTH_TOKEN'}]}'

echo "API Server: $BRIDGE_K8S_MODE_OFF_CLUSTER_ENDPOINT"
echo "Console Image: $CONSOLE_IMAGE"
echo "Console URL: http://localhost:${CONSOLE_PORT}"
echo "Console Platform: $CONSOLE_IMAGE_PLATFORM"

# Prefer podman if installed. Otherwise, fall back to docker.
if [ -x "$(command -v podman)" ]; then
    if [ "$(uname -s)" = "Linux" ]; then
        # Use host networking on Linux since host.containers.internal is unreachable in some environments.
        export BRIDGE_PLUGINS="${PLUGIN_NAME}=http://localhost:9001"
        podman run --pull always --platform "$CONSOLE_IMAGE_PLATFORM" --rm --network=host --env-file <(printenv | grep ^BRIDGE_) "$CONSOLE_IMAGE"
    else
        export BRIDGE_PLUGINS="${PLUGIN_NAME}=http://host.containers.internal:9001"
        podman run --pull always --platform "$CONSOLE_IMAGE_PLATFORM" --rm -p "$CONSOLE_PORT":9000 --env-file <(printenv | grep ^BRIDGE_) "$CONSOLE_IMAGE"
    fi
else
    export BRIDGE_PLUGINS="${PLUGIN_NAME}=http://host.docker.internal:9001"
    docker run --pull always --platform "$CONSOLE_IMAGE_PLATFORM" --rm -p "$CONSOLE_PORT":9000 --env-file <(printenv | grep ^BRIDGE_) "$CONSOLE_IMAGE"
fi

#!/bin/bash
#
# File modified from https://github.com/GoogleCloudPlatform/marketplace-k8s-app-tools/blob/master/marketplace/deployer_util/deploy.sh
#
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eox pipefail

# This is the entry point for the production deployment

# If any command returns with non-zero exit code, set -e will cause the script
# to exit. Prior to exit, set App assembly status to "Failed".
handle_failure() {
  code=$?
  if [[ -z "$NAME" ]] || [[ -z "$NAMESPACE" ]]; then
    # /bin/expand_config.py might have failed.
    # We fall back to the unexpanded params to get the name and namespace.
    NAME="$(/bin/print_config.py \
            --xtype NAME \
            --values_mode raw)"
    NAMESPACE="$(/bin/print_config.py \
            --xtype NAMESPACE \
            --values_mode raw)"
    export NAME
    export NAMESPACE
  fi
  patch_assembly_phase.sh --status="Failed"
  exit $code
}
trap "handle_failure" EXIT

get_token() {
    resource="$(kubectl -n "$NAMESPACE" get secret | grep "${NAME}-svcacct-token" | head -n1 | awk '{print $1}')"
    token="$(kubectl -n "$NAMESPACE" get secret "$resource" -o jsonpath='{.data.token}' | base64 -d)"
    if [[ -z "$token" ]]; then
        echo "Kube token could not be obtained for ${resource}" 1>&2
        return 1
    fi
    echo "Kube token was obtained for ${resource}" 1>&2
    echo "$token"
}

not_deploying_to_stackrox_namespace() {
    [[ "$(cat /data/values/namespace)" != stackrox ]]
}

update_description() {
    name="$1"
    connect="$2"

    # Template and update description.
    markdown="$(kubectl -n stackrox get application "$name" -o jsonpath='{.spec.descriptor.description}')"
    markdown="$(echo "$markdown" | CONNECT="$connect" envsubst | jq --slurp --raw-input .)"
    kubectl -n stackrox patch application "$name" --type=json \
        -p "[{\"op\":\"replace\",\"path\":\"/spec/descriptor/description\",\"value\":${markdown}}]"
}

update_network_docs() {
    local name="$(cat /data/values/name)"
    local network_type="$(cat /data/values/network)"
    local selector
    local ip_address
    local text

    # Use the appropriate selector.
    case "$network_type" in
        "Load Balancer")
            selector='{.status.loadBalancer.ingress[0].ip}'
            ;;
        "Node Port")
            selector='{.spec.clusterIP}'
            ;;
        "None")
            update_description "$name" ''
            return
            ;;
    esac

    # Wait for service to be ready for 60 seconds.
    echo "Waiting for service address"
    for i in $(seq 1 24); do
        ip_address="$(kubectl -n stackrox get svc central-loadbalancer -o jsonpath="$selector")"
        if [[ -n "$ip_address" ]]; then
            break
        fi
        sleep 5
    done

    # Bail out if the service wasn't ready in time.
    if [[ -z "$ip_address" ]]; then
        return
    fi

    # Build text for application info.
    case "$network_type" in
        "Load Balancer")
            text="https://${ip_address}/login"
            update_description "$name" "In a browser, [visit ${text}](${text}) to access StackRox."
            ;;
        "Node Port")
            text="${ip_address}:443"
            update_description "$name" ''
            ;;
    esac

    # Update application info.
    echo "Got address ${text}"
    kubectl -n stackrox patch application "$name" --type=json \
    -p "[{\"op\":\"add\",\"path\":\"/spec/info/-\",\"value\":{\"name\":\"Stackrox address\",\"value\":\"${text}\"}}]"
}

NAME="$(/bin/print_config.py \
    --xtype NAME \
    --values_mode raw)"
NAMESPACE="$(/bin/print_config.py \
    --xtype NAMESPACE \
    --values_mode raw)"
export NAME
export NAMESPACE

# Obtain service account token and assume identity
export KUBE_TOKEN="$(get_token)"

# Create and check for the stackrox namespace
kubectl create namespace stackrox || true
kubectl get namespace stackrox
export NAMESPACE=stackrox

echo "Deploying application \"$NAME\""

if not_deploying_to_stackrox_namespace; then
    cat /data/application.yaml.tpl | envsubst > /data/application.yaml
    kubectl apply --namespace="$NAMESPACE" -f /data/application.yaml
fi

app_uid=$(kubectl get "applications.app.k8s.io/$NAME" \
  --namespace="$NAMESPACE" \
  --output=jsonpath='{.metadata.uid}')
app_api_version=$(kubectl get "applications.app.k8s.io/$NAME" \
  --namespace="$NAMESPACE" \
  --output=jsonpath='{.apiVersion}')

/bin/expand_config.py --values_mode raw --app_uid "$app_uid"

roxctl gcp generate --values-file /data/final_values.yaml --output-dir /tmp/stackrox
mkdir -p /data/chart
mv /tmp/stackrox/central /tmp/stackrox/chart
tar -czvf /data/chart/central.tar.gz -C /tmp/stackrox chart

create_manifests.sh

# Assign owner references for the resources.
/bin/set_ownership.py \
  --app_name "$NAME" \
  --app_uid "$app_uid" \
  --app_api_version "$app_api_version" \
  --manifests "/data/manifest-expanded" \
  --dest "/data/resources.yaml"

# Ensure assembly phase is "Pending", until successful kubectl apply.
/bin/setassemblyphase.py \
  --manifest "/data/resources.yaml" \
  --status "Pending"

# Apply the manifest.
kubectl apply --namespace="$NAMESPACE" --filename="/data/resources.yaml"

update_network_docs

# Clean up IAM resources
patch_assembly_phase.sh --status="Success"
export NAMESPACE="$(cat /data/values/namespace)"

if not_deploying_to_stackrox_namespace; then
    cat /data/application-success.yaml.tpl | envsubst > /data/application-success.yaml
    kubectl patch --namespace="$NAMESPACE" application "$NAME" --type merge --patch "$(cat /data/application-success.yaml)"
fi

kubectl -n "$NAMESPACE" delete serviceaccount "${NAME}-deployer-sa"
kubectl -n "$NAMESPACE" delete rolebinding    "${NAME}-deployer-rb"
kubectl -n "$NAMESPACE" delete serviceaccount "${NAME}-svcacct"

trap - EXIT

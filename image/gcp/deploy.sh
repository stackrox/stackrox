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
if ! type jq >/dev/null 2>&1; then
	echo "Error: jq not found on your system. Please install jq and rerun this script." 1>&2
	exit 1
fi

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
    resource="$(kubectl -n "$STACKROX_NAMESPACE" get secret | grep "${NAME}-svcacct-token" | head -n1 | awk '{print $1}')"
    token="$(kubectl -n "$STACKROX_NAMESPACE" get secret "$resource" -o jsonpath='{.data.token}' | base64 -d)"
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
    kubectl -n "$STACKROX_NAMESPACE" patch application "$name" --type=json \
        -p "[{\"op\":\"replace\",\"path\":\"/spec/descriptor/description\",\"value\":${markdown}}]"
}

wait_for_central_pod() {
    # Wait 2 minutes for a central pod to be ready.
    echo "Waiting for central pod..."
    kubectl -n "$STACKROX_NAMESPACE" wait --for=condition=Ready --selector=app=central --timeout=2m pod
}

wait_for_sensor_pod() {
    # Wait 2 minutes for a sensor pod to be ready.
    echo "Waiting for sensor pod..."
    kubectl -n "$STACKROX_NAMESPACE" wait --for=condition=Ready --selector=app=sensor --timeout=2m pod
}

wait_for_central_api() {
    # Wait 2 minutes for the central API to be ready.
    echo "Waiting for central api..."
    for i in $(seq 1 24); do
        status="$( (curl --max-time 5 -ks https://central.stackrox:443/v1/ping || true) | jq -r .status)"
        if [[ "$status" = 'ok' ]]; then
            echo "API ready"
            break
        fi
        sleep 5
    done
}

run_compliance() {
    password="$(cat /tmp/stackrox/password)"
    endpoint='central.stackrox:443'
    compliance_body='{"selection":{"cluster_id":"*","standard_id":"*"}}'

    echo "Running compliance scans..."
    curl -u "admin:${password}" -k -XPOST -d "$compliance_body"    "https://${endpoint}/v1/compliancemanagement/runs"
}

get_cluster_name() {
    # Get the name of the first listed node. Node name should be in the form
    # "gke-<NAME>-default-pool-0b062835-7gfz"
    node_name="$(kubectl get nodes -o jsonpath={.items[0].metadata.name} 2>/dev/null || true)"
    if [[ -z "$node_name" ]]; then
        echo "gke"
        return
    fi

    # Extract the cluster name from the middle of the node name.
    cluster_name="$(echo "$node_name" | sed 's/gke-\(.*\)-.*-pool.*/\1/')"
    if [[ -z "$node_name" ]]; then
        echo "gke"
        return
    fi

    echo "$cluster_name" | tr '[:upper:]' '[:lower:]'
}

try_install_sensor() {
    if [[ -z "$(cat /data/values/license)" ]]; then
        echo "StackRox isn't currently licensed, cannot create sensor."
        return 0
    fi

    collection_method="none"
    collector_image=""
    if [[ -n "$(cat /data/values/stackrox-io-username)" ]] && [[ -n "$(cat /data/values/stackrox-io-password)" ]]; then
        echo "Deploying collector from stackrox.io."
        export REGISTRY_USERNAME="$(cat /data/values/stackrox-io-username)"
        export REGISTRY_PASSWORD="$(cat /data/values/stackrox-io-password)"
        collection_method="kernel-module"
        collector_image="collector.stackrox.io/collector"
    fi

    # Get cluster name, if possible.
    cluster_name="$(get_cluster_name)"

    wait_for_central_pod
    wait_for_central_api

    main_image="$(cat /data/values/main-image)"
    gcp_version=""
    if [[ $main_image =~ [0-9]*\.[0-9]*$ ]]; then
        gcp_version=":$BASH_REMATCH"
    fi

    roxctl --endpoint central.stackrox:443 --password "$(cat /tmp/stackrox/password)" sensor generate k8s \
    --central central.stackrox:443 \
    --collection-method "$collection_method" \
    --collector-image-repository "$collector_image" \
    --main-image-repository "$(echo $main_image | sed "s,[:@].*,$gcp_version,")" \
    --name "$cluster_name"

    if [[ "$?" -ne 0 ]]; then
        echo "Failed to provision sensor with roxctl."
        return 0
    fi

    "./sensor-${cluster_name}/sensor.sh"

    wait_for_sensor_pod
    run_compliance
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
            selector='{.status.loadBalancer.ingress..ip}'
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
        ip_address="$(kubectl -n "$STACKROX_NAMESPACE" get svc central-loadbalancer -o jsonpath="$selector")"
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
    kubectl -n "$STACKROX_NAMESPACE" patch application "$name" --type=json \
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
export STACKROX_NAMESPACE=stackrox

echo "Deploying application \"$NAME\""

app_uid=$(kubectl get "applications.app.k8s.io/$NAME" \
  --namespace="$NAMESPACE" \
  --output=jsonpath='{.metadata.uid}')
app_api_version=$(kubectl get "applications.app.k8s.io/$NAME" \
  --namespace="$NAMESPACE" \
  --output=jsonpath='{.apiVersion}')

/bin/expand_config.py --values_mode raw --app_uid "$app_uid"

# Get the API versions from the API server.
api_version_args=()

while IFS='' read -r line || [[ -n "$line" ]]; do
  [[ -n "$line" ]] || continue
  api_version_args+=(-a "$line")
done < <({ kubectl get --raw /apis | jq -r '.groups[].versions[].groupVersion | ("/apis/" + .)' ; echo /api/v1 ; } | xargs -n 1 kubectl get --raw | jq '([.groupVersion, (.groupVersion | sub("/.*$"; ""))] | unique) + [.groupVersion + "/" + .resources[].kind] | .[]' -r)

# Generate installation bundles for both Central and Scanner.
roxctl gcp generate --values-file /data/final_values.yaml --output-dir /tmp/stackrox
mkdir -p /tmp/stackrox/rendered
helm template -n "$STACKROX_NAMESPACE" stackrox-central-services /tmp/stackrox/chart/ \
  -f /tmp/stackrox/values-public.yaml -f /tmp/stackrox/values-private.yaml \
  --set imagePullSecrets.allowNone=true \
  "${api_version_args[@]}" \
  >/tmp/stackrox/rendered/resources.yaml

## Assign owner references for the resources.
/bin/set_ownership.py \
  --app_name "$NAME" \
  --app_uid "$app_uid" \
  --app_api_version "$app_api_version" \
  --manifests /tmp/stackrox/rendered/resources.yaml \
  --dest /data/resources.yaml

# Ensure assembly phase is "Pending", until successful kubectl apply.
/bin/setassemblyphase.py \
  --manifest "/data/resources.yaml" \
  --status "Pending"

# Apply the manifest.
kubectl apply -n "$STACKROX_NAMESPACE" --filename=/data/resources.yaml

# Attempt to provision a sensor.
try_install_sensor || true

# Attempt to update the GKE Application sidebar docs.
update_network_docs || true

# Clean up IAM resources
patch_assembly_phase.sh --status="Success"
export NAMESPACE="$(cat /data/values/namespace)"

cat /data/application.yaml.tpl | envsubst > /data/application.yaml
kubectl apply --namespace="$NAMESPACE" -f /data/application.yaml

kubectl -n "$NAMESPACE" delete serviceaccount "${NAME}-deployer-sa"
kubectl -n "$NAMESPACE" delete rolebinding    "${NAME}-deployer-rb"
kubectl -n "$NAMESPACE" delete serviceaccount "${NAME}-svcacct"

trap - EXIT

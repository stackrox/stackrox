#!/usr/bin/env bash
set -euo pipefail
set -x

# Dir with code used for deployments
dir="$1"
# Dir with * migrate-to-operator code
dirmig="$2"

# Identify commit that includes fix(operator): wipe fields when migrating to operator
tag=$(make --no-print-directory -C "$dir" tag)
# Cleanup previous install
rm -f crs1.yaml cr-central.yaml cr-sensor.yaml
rm -rf deploy-central deploy-sensor
oc get ns
oc delete -n stackrox centrals.platform.stackrox.io stackrox-central-services || true
oc delete -n stackrox securedclusters.platform.stackrox.io stackrox-secured-cluster-services || true
oc delete ns stackrox || true
helm uninstall -n rhacs-operator-system rhacs-operator || true
oc delete customresourcedefinitions.apiextensions.k8s.io centrals.platform.stackrox.io securedclusters.platform.stackrox.io securitypolicies.config.stackrox.io || true
# Get roxctl from that commit
make -C "$dir" cli_host-arch
roxctl="$dir/bin/linux_amd64/roxctl"
# Deploy central from roxctl
central_dir="deploy-central"
$roxctl central generate k8s pvc --output-dir $central_dir --db-name='central-dbx'
cd $central_dir
central/scripts/setup.sh
kubectl create -R -f central
scanner/scripts/setup.sh
kubectl create -R -f scanner
scanner-v4/scripts/setup.sh
kubectl create -R -f scanner-v4
cd -
kubectl get -n stackrox pod

kubectl -n stackrox rollout status deployment central

# Port forward
nohup oc port-forward -n stackrox svc/central "8443:443" --address='0.0.0.0' 1>/dev/null 2>&1 &
sleep 2

export API_ENDPOINT=localhost:8443
export ROX_ADMIN_PASSWORD
ROX_ADMIN_PASSWORD="$(cat $central_dir/password)"

#CRS
#$roxctl central crs generate crs1 --output crs1.yaml --insecure-skip-tls-verify=true
#oc -n stackrox apply -f crs1.yaml

# Get bundle
sensor_dir="deploy-sensor"
$roxctl --insecure-skip-tls-verify=true sensor generate k8s --name same --output-dir $sensor_dir

# Deploy sc from roxctl bundle
$sensor_dir/sensor.sh

# Wait for health
kubectl -n stackrox rollout status daemonset collector
kubectl -n stackrox rollout status deployment sensor
kubectl -n stackrox rollout status deployment admission-control

dummy=""
while [[ -z $dummy ]]; do
  ( . scripts/lib.sh; roxcurl /v1/clusters | jq '.clusters[]|(.name,.managedBy,.healthStatus)' )
  echo y+ENTER when ok
  read dummy
done

# Build chart from that commit
ROX_OPERATOR_SKIP_PROTO_GENERATED_SRCS=true $dir/operator/hack/generate-chart.sh development_build

# Install operator using chart --take-ownership (config CRS)
ROX_NAMESPACE=rhacs-operator-system $central_dir/central/scripts/setup.sh
helm install --take-ownership --wait -n rhacs-operator-system \
 --set manager.imagePullSecrets[0].name=stackrox rhacs-operator $dir/operator/dist/chart/

# Get migration command roxctl
make -C "$dirmig" cli_host-arch
migroxctl="$dirmig/bin/linux_amd64/roxctl"

# Convert to central, apply it
$migroxctl central migrate-to-operator --namespace stackrox > cr-central.yaml
cat cr-central.yaml
oc apply -n stackrox -f cr-central.yaml
sleep 5

# Wait for health
kubectl -n stackrox rollout status deployment central
# Port forward
nohup oc port-forward -n stackrox svc/central "8443:443" --address='0.0.0.0' 1>/dev/null 2>&1 &
sleep 2

( . scripts/lib.sh; roxcurl /v1/clusters | jq '.clusters[]|(.name,.healthStatus)' )
echo ENTER when ok
read dummy

# Convert to sc, apply it
$migroxctl sensor migrate-to-operator --namespace stackrox > cr-sensor.yaml
cat cr-sensor.yaml
oc apply -n stackrox -f cr-sensor.yaml
sleep 5

# Wait for health
kubectl -n stackrox rollout status daemonset collector
kubectl -n stackrox rollout status deployment sensor
kubectl -n stackrox rollout status deployment admission-control
dummy=""
while [[ -z $dummy ]]; do
  ( . scripts/lib.sh; roxcurl /v1/clusters | jq '.clusters[]|(.name,.managedBy,.healthStatus)' )
  echo y+ENTER when ok
  read dummy
done


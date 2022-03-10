#!/bin/bash
# vim: set sw=4 et:
set -eu
exit 1

# Hoisting from CI to here... figuring out what parts are needed to bringup
# a cluster and provision test environment, fixtures, etc and run a single
# test. Actual code needed for this is in the `stepX-*.sh` scripts and I'm
# dumping misc snippets and stuff I find here to sort it out.


function stackrox_secure_cluster {
    # Stackrox License Activation
    ROX_LICENSE_FILE="/tmp/STACKROX_LICENSE_20210312.txt"  # roxctl uses this env var
    pass STACKROX_LICENSE_20210312 > "$ROX_LICENSE_FILE"
    ROX_CENTRAL_ADDRESS="$(kubectl -n "$STACKROX_NAMESPACE" get svc central -o json \
        | jq -r '.status.loadBalancer.ingress[].ip'):443"
    #ROX_CENTRAL_ADMIN_PASSWORD=$(cat $CENTRAL_BUNDLE_DPATH/password)
    ROX_CENTRAL_ADMIN_PASSWORD=$(cat deploy/openshift/central-deploy/password)
    ROXCTL="roxctl -e $ROX_CENTRAL_ADDRESS -p $ROX_CENTRAL_ADMIN_PASSWORD --insecure-skip-tls-verify"
    $ROXCTL central whoami
    $ROXCTL central license add --license=@"$ROX_LICENSE_FILE"

    # Verify Central Deployment
    kubectl get pod -n "$STACKROX_NAMESPACE" -w
    pbcopy < "$CENTRAL_BUNDLE_DPATH/password"
    open "https://$ROX_CENTRAL_ADDRESS"

    # Deploy Sensor
    kubectl config use-context "$KUBE_CONTEXT_2"
    SENSOR_RESOURCE_CONFIG_DIR="$CENTRAL_BUNDLE_DPATH/sensor-$TEST_CLUSTER_2"
    rm -rf "$SENSOR_RESOURCE_CONFIG_DIR"
    $ROXCTL sensor generate k8s --help | view -

    # TODO: add specific options for kernel module, etc
    $ROXCTL sensor generate k8s --name "$TEST_CLUSTER_2" --central "$ROX_CENTRAL_ADDRESS"
    $ROXCTL sensor get-bundle "$TEST_CLUSTER_2"
    "$CENTRAL_BUNDLE_DPATH/sensor-$TEST_CLUSTER_2/sensor.sh"

    kubectl config use-context "$KUBE_CONTEXT_1"
    echo "Create cluster monitor configuration via the web console with eBPF based monitoring"
    echo "Downlaod and unpack the .zip archive anr run the sensor.sh script to install"
    cd /tmp && ./sensor-upgrade-test1-3-0-57-0-rc-3/sensor.sh

    # Stackrox Test
    kubectl config use-context "$KUBE_CONTEXT_2"
    kubectl create ns test
    kubectl run shell --labels=app=shellshock,team=test-team --image=vulnerables/cve-2014-6271 -n test
    kubectl run samba --labels=app=rce --image=vulnerables/cve-2017-7494 -n test

    # Roxctl basic functional verification
    $ROXCTL deployment check \
      --file="$CENTRAL_BUNDLE_DPATH/sensor-$TEST_CLUSTER_2/sensor.yaml" \
      --insecure-skip-tls-verify
    $ROXCTL image check --image="nginx"
    $ROXCTL image scan --image="nginx" --force
}

###############################################################################
# Setup default TLS certificates
cert_dir="$(mktemp -d)"
./tests/scripts/setup-certs.sh "$cert_dir" custom-tls-cert.central.stackrox.local "Server CA"
export ROX_DEFAULT_TLS_CERT_FILE="${cert_dir}/tls.crt"
export ROX_DEFAULT_TLS_KEY_FILE="${cert_dir}/tls.key"
export DEFAULT_CA_FILE="${cert_dir}/ca.crt"
ROX_TEST_CA_PEM=$(cat "${cert_dir}/ca.crt")
export ROX_TEST_CA_PEM
export ROX_TEST_CENTRAL_CN="custom-tls-cert.central.stackrox.local"
export TRUSTSTORE_PATH="${cert_dir}/keystore.p12"
echo "contents of ${cert_dir}:"
ls -al "${cert_dir}"

###############################################################################
# Deploy Stackrox

# Periodically log cluster info in case it helps for troubleshooting
scripts/ci/deployment-minder.sh &> "$SCRATCH/log/deployment-minder.log" &

export REGISTRY_USERNAME="$DOCKER_IO_PULL_USERNAME"  # used by tests
export REGISTRY_PASSWORD="$DOCKER_IO_PULL_PASSWORD"  # used by tests

./deploy/k8s/central.sh
#./deploy/openshift/central.sh

read -r ROX_PASSWORD <<< "$(cat ./deploy/openshift/central-deploy/password)"
read -r ROX_USERNAME <<< "admin"
export ROX_USERNAME ROX_PASSWORD

###############################################################################
# Wait for Central API
echo "waiting for central api"
start_seconds="$(date '+%s')"

while true; do
  kubectl -n stackrox get deploy/central -o json /tmp/central.json
  central_replicas=$(jq '.status.replicas' /tmp/central.json) || true
  central_ready_replicas=$(jq '.status.readyReplicas' /tmp/central.json) || true

  if [[ "$central_replicas" -eq 1 ]]; then
    if [[ "$central_ready_replicas" -eq 1 ]]; then
      break
    fi
  fi

  cat /tmp/central.json

  now_seconds=$(date '+%s')
  elapsed_seconds=$(( now_seconds - start_seconds ))
  if (( elapsed_seconds > 300 )); then
    kubectl -n stackrox get pod -o wide
    kubectl -n stackrox get deploy -o wide
    bash_exit_failure "Timed out after 5m"
  else
    echo "start_seconds   => [$start_seconds]"
    echo "now_seconds     => [$now_seconds]"
    echo "elapsed_seconds => [$elapsed_seconds]"
  fi

  echo -n .
  sleep 10
done
echo "central is running"

API_HOSTNAME=localhost
API_PORT=8000
if [[ "$LOAD_BALANCER" == "lb" ]]; then
  API_HOSTNAME=$(./scripts/k8s/get-lb-ip.sh)
  export API_HOSTNAME
  export API_PORT=443
fi
API_ENDPOINT="$API_HOSTNAME:$API_PORT"
METADATA_URL="https://$API_ENDPOINT/v1/metadata"
export API_HOSTNAME API_PORT API_ENDPOINT METADATA_URL

echo "API_HOSTNAME => $API_HOSTNAME"
echo "API_PORT     => $API_PORT"
echo "API_ENDPOINT => $API_ENDPOINT"
echo "METADATA_URL => $METADATA_URL"

(( hit_count=0 ))
for idx in $(seq 1 20); do
  curl -sk --connect-timeout 5 --max-time 10 "$METADATA_URL" > /tmp/metadata.json || true
  license_status=$(jq -r '.licenseStatus' /tmp/metadata.json) || true
  echo "[$idx] license_status: [$license_status]"

  if grep "RESTARTING" <<< "$license_status"; then
    sleep 5
    continue
  fi

  if grep "." <<< "$license_status"; then
    (( hit_count+=1 ))
  fi
  echo "hit_count is $hit_count"
  if [[ "$hit_count" -eq "3" ]]; then
    break
  fi

  sleep 5
done

kubectl -n stackrox get pods

if [[ "$hit_count" -lt "3" ]]; then
  bash_exit_failure "Failed to connect to Central"
fi

###############################################################################
# Deploy Sensor
# Sensor is CPU starved under OpenShift causing all manner of test failures.
# * https://stack-rox.atlassian.net/browse/ROX-5334
# * https://stack-rox.atlassian.net/browse/ROX-6891
echo "Deploying Sensor using Helm ..."
export SENSOR_HELM_DEPLOY=true
export ADMISSION_CONTROLLER=true

deploy/openshift/sensor.sh
kubectl -n stackrox set resources deploy/sensor -c sensor --requests 'cpu=2' --limits 'cpu=4'
./scripts/ci/sensor-wait.sh

# Bounce collectors to avoid restarts on initial module pull
kubectl -n stackrox delete pod -l app=collector --grace-period=0


function stackrox_deploy_via_roxctl_config_generation {
  # Untested. Is this even needed?
  cd /tmp
  rm -rf "$CENTRAL_BUNDLE_DPATH"  # roxctl will recreate the dir
  roxctl central generate interactive  # -> k8s, lb, pvc, helm, defaults
  cd "$CENTRAL_BUNDLE_DPATH"
}

# The deploy.sh script deploy central and sensor and we get a fully working cluster.
# So this next step might be unnecessary.
# Maybe needed for tasks under 'build_all' circleci target?
if false; then
  stackrox_secure_cluster
fi

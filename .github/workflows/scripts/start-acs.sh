#!/usr/bin/bash
set -euox pipefail


cd $STACKROX_DIR

echo "GITHUB_OUTPUT= ${GITHUB_OUTPUT}"
./deploy/k8s/central.sh
kubectl -n stackrox port-forward deploy/central 8000:8443 > /dev/null 2>&1 &
sleep 20

./deploy/k8s/sensor.sh

kubectl -n stackrox set env deploy/sensor MUTEX_WATCHDOG_TIMEOUT_SECS=0 ROX_FAKE_KUBERNETES_WORKLOAD=long-running ROX_FAKE_WORKLOAD_STORAGE=/var/cache/stackrox/pebble.db
kubectl -n stackrox patch deploy/sensor -p '{"spec":{"template":{"spec":{"containers":[{"name":"sensor","resources":{"requests":{"memory":"3Gi","cpu":"2"},"limits":{"memory":"12Gi","cpu":"4"}}}]}}}}'

kubectl -n stackrox set env deploy/central MUTEX_WATCHDOG_TIMEOUT_SECS=0
kubectl -n stackrox patch deploy/central -p '{"spec":{"template":{"spec":{"containers":[{"name":"central","resources":{"requests":{"memory":"3Gi","cpu":"2"},"limits":{"memory":"12Gi","cpu":"4"}}}]}}}}'

max_tries=200
tries=0
while [ "$tries" -lt "$max_tries" ]; do
    ready="$(kubectl -n stackrox get pod -l app=central -o jsonpath='{.items[*].status.containerStatuses[?(@.ready == true)]}')"

    if [[ -n "$ready" ]]; then
	    echo "Central is running"
	    break
    fi

    tries=$((tries + 1))
    sleep 1
done

sleep 60

ROX_ADMIN_PASSWORD=$(cat deploy/k8s/central-deploy/password)
echo "::add-mask::$ROX_ADMIN_PASSWORD"
CENTRAL_IP=$(kubectl -n stackrox get svc/central-loadbalancer -o json | jq -r '.status.loadBalancer.ingress[0] | .ip // .hostname')
kubectl -n stackrox create secret generic access-rhacs --from-literal="username=${ROX_ADMIN_USERNAME}" --from-literal="password=${ROX_ADMIN_PASSWORD}" --from-literal="central_url=https://${CENTRAL_IP}"
echo "rox_password=${ROX_ADMIN_PASSWORD}" >> "$GITHUB_OUTPUT"
echo "cluster_name=${NAME//./-}" >> "$GITHUB_OUTPUT"

printf "Long-running GKE cluster %s has been patched.\nAccess it by running \`./scripts/release-tools/setup-central-access.sh %s\` from your local machine." "${NAME//./-}" "${NAME//./-}" >> "$GITHUB_STEP_SUMMARY"

echo "central-password='${ROX_ADMIN_PASSWORD}'" >> "$GITHUB_OUTPUT"

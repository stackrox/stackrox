#!/usr/bin/env bash
set -eou pipefail

ROX_ENDPOINT=${1:-https://localhost:8000}

get_process_baseline() {
  query="key.deploymentId=${deployment_id}&key.containerName=${container_name}&key.clusterId=${cluster_id}&key.namespace=${namespace}"
  
  process_baseline_json="$(curl --location --silent --request GET "${ROX_ENDPOINT}/v1/processbaselines/key?${query}" -k --header "Authorization: Bearer $ROX_API_TOKEN")"
  
  echo "$process_baseline_json" | jq
}

get_processes() {
  container_json="$(curl --location --silent --request GET "${ROX_ENDPOINT}/v1/processes/deployment/${deployment_id}/grouped/container" -k --header "Authorization: Bearer $ROX_API_TOKEN")"

  echo "$container_json" | jq
}

get_violations() {
  violations_json="$(curl --location --silent --request GET "${ROX_ENDPOINT}/v1/alerts" -k --header "Authorization: Bearer $ROX_API_TOKEN")"
  
  ubuntu_violations_json="$(echo "$violations_json" | jq '.alerts[] | select(.deployment.id == "'"$deployment_id"'")')"
  process_violations_json="$(echo "$ubuntu_violations_json" | jq 'select(.policy.name == "Unauthorized Process Execution")')"
  
  violation_id="$(echo "$process_violations_json" | jq -r .id)"
  
  detailed_violation_json="$(curl --location --silent --request GET "${ROX_ENDPOINT}/v1/alerts/${violation_id}" -k --header "Authorization: Bearer $ROX_API_TOKEN")"

  echo "$detailed_violation_json" | jq
}

lock_process_baseline() {
  data="$(echo "$key" | jq '{
	  keys: [
	  		.
		],
		locked: true
	}')"

  process_baselines_json="$(curl --location --silent --request PUT "${ROX_ENDPOINT}/v1/processbaselines/lock" -k --header "Authorization: Bearer $ROX_API_TOKEN" --data "$data")"
}

unlock_process_baseline() {
  data="$(echo "$key" | jq '{
	  keys: [
	  		.
		],
		locked: false
	}')"

  process_baselines_json="$(curl --location --silent --request PUT "${ROX_ENDPOINT}/v1/processbaselines/lock" -k --header "Authorization: Bearer $ROX_API_TOKEN" --data "$data")"
}

get_state() {
  echo "Process baseline"
  process_baseline_json="$(get_process_baseline)"
  echo "$process_baseline_json" | jq
  echo
  echo
  echo
  
  container_json="$(get_processes)"
  
  echo "Processes"
  echo "$container_json" | jq
  echo
  echo
  echo
  
  violations_json="$(get_violations)"
  
  echo "Violations"
  echo "$violations_json" | jq
  echo
  echo
  echo
}

header() {
  echo
  echo "############################################"
  echo
}

wait_time=20

kubectl delete pod ubuntu-pod || true

echo "Creating ubuntu-pod deployment"
kubectl run ubuntu-pod --image=ubuntu --restart=Never --command -- sleep infinity
kubectl wait --for=condition=Ready pod/ubuntu-pod --timeout=300s

kubectl exec ubuntu-pod -it -- cat /proc/1/net/tcp

sleep "$wait_time"

json_deployments="$(curl --location --silent --request GET "${ROX_ENDPOINT}/v1/deploymentswithprocessinfo" -k -H "Authorization: Bearer $ROX_API_TOKEN")"

echo "Initial deployments with process info"
echo "$json_deployments" | jq
echo
echo
echo

json_keys="$(echo $json_deployments | jq '{
                       keys: [.deployments[] | .deployment as $d | select(.deployment.name == "ubuntu-pod") | {
                         deployment_id: $d.id,
                         container_name: "ubuntu-pod",
                         cluster_id: $d.clusterId,
                         namespace: "default"
                       }],
                     }')"

key="$(echo $json_keys | jq .keys[0])"
echo "$key" | jq

deployment_id="$(echo $key | jq -r .deployment_id)"
container_name="$(echo $key | jq -r .container_name)"
cluster_id="$(echo $key | jq -r .cluster_id)"
namespace="$(echo $key | jq -r .namespace)"

header
echo "Initial state"
get_state

echo "Sleep for three minutes"
sleep 3m
echo "Plus a buffer"
sleep 30s

header
echo "After sleep"
get_state

kubectl exec ubuntu-pod -it -- tac /proc/1/net/tcp
sleep "$wait_time"

header
echo "After tac"
get_state

unlock_process_baseline

sleep "$wait_time"

header
echo "After unlocking process baseline"
get_state

kubectl exec ubuntu-pod -it -- ls /proc/1/net/tcp
sleep "$wait_time"

header
echo "After running a process after unlocking"
get_state

lock_process_baseline

sleep "$wait_time"

header
echo "After manually locking"
get_state

kubectl exec ubuntu-pod -it -- basename /proc/1/net/tcp
sleep "$wait_time"

header
echo "After running a process after manually locking"
get_state

echo "Completed script"

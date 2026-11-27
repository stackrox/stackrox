#!/bin/bash
# Scanner V4 DB reinitializes on version downgrade (ROX-36223).
# NAMESPACE is supplied by KUTTL.

set -euo pipefail

NAMESPACE=${NAMESPACE:-}
if [[ -z "$NAMESPACE" ]]; then
    echo >&2 "NAMESPACE must be set."
    exit 1
fi

central=stackrox-central-services
version_file=/var/lib/postgresql/data/pgdata/.scanner-db-version

kubectl_in_namespace() {
    retry-kubectl.sh </dev/null -n "$NAMESPACE" "$@"
}

echo "Checking the Scanner V4 DB version file"
kubectl_in_namespace rollout status deploy/scanner-v4-db --timeout=600s
original_version=$(kubectl_in_namespace exec deploy/scanner-v4-db -c db -- cat "$version_file")
[[ "$original_version" =~ ^[0-9]+\.[0-9]+$ ]]
original_build_tag=$(kubectl_in_namespace get deploy/scanner-v4-db \
    -o jsonpath='{.spec.template.spec.initContainers[?(@.name=="init-db")].env[?(@.name=="BUILD_TAG")].value}')
[[ -n "$original_build_tag" ]]

set_build_tag() {
    local tag="$1"
    kubectl_in_namespace patch deploy/scanner-v4-db --type=strategic \
        -p "{\"spec\":{\"template\":{\"spec\":{\"initContainers\":[{\"name\":\"init-db\",\"env\":[{\"name\":\"BUILD_TAG\",\"value\":\"${tag}\"}]}]}}}}"
}

cleanup() {
    local result=$?
    trap - EXIT
    # Restore the override even if the test failed, and always resume reconciliation.
    set_build_tag "$original_build_tag" || result=1
    kubectl_in_namespace annotate central "$central" stackrox.io/pause-reconcile- || result=1
    kubectl_in_namespace rollout status deploy/scanner-v4-db --timeout=600s || result=1
    exit "$result"
}
trap cleanup EXIT

echo "Pausing reconciliation before overriding the Scanner V4 DB version"
kubectl_in_namespace annotate central "$central" stackrox.io/pause-reconcile=true
kubectl_in_namespace wait central/"$central" --for=condition=Paused --timeout=120s

echo "Simulating a downgrade to version 4.11.7"
set_build_tag "4.11.7"
kubectl_in_namespace rollout status deploy/scanner-v4-db --timeout=600s

echo "Verifying downgrade detection and the updated version file"
logs=$(kubectl_in_namespace logs deploy/scanner-v4-db -c init-db)
echo "$logs"
grep -q "downgrade detected" <<<"$logs"
version=$(kubectl_in_namespace exec deploy/scanner-v4-db -c db -- cat "$version_file")
[[ "$version" == "4.11" ]]

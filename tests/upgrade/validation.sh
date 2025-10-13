#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# Test validation functions for upgrades

createPostgresScopes() {
    local scopes=(
    '{"id":"","name":"PostgresScope1","description":"Testing access scope","rules":{"includedClusters":["remote"],"includedNamespaces":[{"clusterName":"remote","namespaceName":"kube-public"},{"clusterName":"remote","namespaceName":"default"}],"clusterLabelSelectors":[],"namespaceLabelSelectors":[]}}'
    '{"id":"","name":"PostgresScope2","description":"Testing access scope","rules":{"includedClusters":["remote"],"includedNamespaces":[{"clusterName":"remote","namespaceName":"kube-public"},{"clusterName":"remote","namespaceName":"default"}],"clusterLabelSelectors":[],"namespaceLabelSelectors":[]}}'
    '{"id":"","name":"PostgresScope3","description":"Testing access scope","rules":{"includedClusters":["remote"],"includedNamespaces":[{"clusterName":"remote","namespaceName":"kube-public"},{"clusterName":"remote","namespaceName":"default"}],"clusterLabelSelectors":[],"namespaceLabelSelectors":[]}}'
        )

        for scopeJSON in "${scopes[@]}"
        do
          tmpOutput=$(mktemp)
          status=$(curl --retry 3 -k --config <(curl_cfg user "admin:${ROX_ADMIN_PASSWORD}") -X POST \
            -d "${scopeJSON}" \
            -o "$tmpOutput" \
            -w "%{http_code}\n" \
            https://"${API_ENDPOINT}"/v1/simpleaccessscopes )

          if [ "${status}" != "200" ] && [ "${status}" != "429" ] && [ "${status}" != "409" ]; then
            cat "$tmpOutput"
            exit 1
          fi
        done
}

checkForPostgresAccessScopes() {
    info "checkForPostgresAccessScopes"
    local accessScopes
    accessScopes=$(curl --retry 3 -sSk -X GET --config <(curl_cfg user "admin:${ROX_ADMIN_PASSWORD}") https://"${API_ENDPOINT}"/v1/simpleaccessscopes)
    echo "access scopes: ${accessScopes}"
    test_equals_non_silent "$(echo "$accessScopes" | jq '.accessScopes[] | select(.name == "PostgresScope1") | .name' -r)" "PostgresScope1"
    test_equals_non_silent "$(echo "$accessScopes" | jq '.accessScopes[] | select(.name == "PostgresScope2") | .name' -r)" "PostgresScope2"
}

verifyNoPostgresAccessScopes() {
    info "verifyNoPostgresAccessScopes"
    local accessScopes
    accessScopes=$(curl --retry 3 -sSk -X GET --config <(curl_cfg user "admin:${ROX_ADMIN_PASSWORD}") https://"${API_ENDPOINT}"/v1/simpleaccessscopes)
    echo "access scopes: ${accessScopes}"
    test_empty_non_silent "$(echo "$accessScopes" | jq '.accessScopes[] | select(.name == "PostgresScope1") | .name' -r)"
    test_empty_non_silent "$(echo "$accessScopes" | jq '.accessScopes[] | select(.name == "PostgresScope2") | .name' -r)"
}

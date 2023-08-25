#!/usr/bin/env bash

# Use tmp directory to install all certs
mkdir -p ./tmp

for cert in "ca.pem" "sensor-cert.pem" "sensor-key.pem"; do
    echo "Fetching $cert"
    cert_data=$(kubectl -n stackrox get secret sensor-tls -o=jsonpath='{$.data}' | jq -r ".[\"$cert\"]" | base64 -d)
    echo "$cert_data" > "./tmp/$cert"
done

echo "Fetching helm config"
config_data=$(kubectl -n stackrox get secret helm-cluster-config -o=jsonpath='{$.data}' | jq -r '.["config.yaml"]' | base64 -d)
echo "$config_data" > "./tmp/helm-config.yaml"

echo "Fetching helm name"
cluster_name=$(kubectl -n stackrox get secret helm-effective-cluster-name -o json | jq -r '.data["cluster-name"]' | base64 -d)
echo "$cluster_name" > "./tmp/helm-name.yaml"

fingerprint=$(echo "$config_data" | grep "configFingerprint" | xargs | awk '{print $2}')
echo "Helm Fingerprint: $fingerprint"
echo "Run the following command in your shell to export helm fingerprint:"
echo "export ROX_HELM_CLUSTER_CONFIG_FP=$fingerprint"


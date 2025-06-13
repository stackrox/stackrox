#!/bin/bash

mkdir -p certs
echo export ROX_SENSOR_ENDPOINT=sensor.default.svc:443
echo export ROX_MTLS_CERT_FILE=$PWD/certs/cert.pem
echo export ROX_MTLS_KEY_FILE=$PWD/certs/key.pem
echo export ROX_MTLS_CA_FILE=$PWD/certs/ca.pem
kubectl get secret tls-cert-admission-control -o json | jq -r '.data["cert.pem"]' | base64 -d > certs/cert.pem
kubectl get secret tls-cert-admission-control -o json | jq -r '.data["key.pem"]' | base64 -d > certs/key.pem
kubectl get secret additional-ca -o json | jq -r '.data["ca.pem"]' | base64 -d > certs/ca.pem

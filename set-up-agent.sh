#!/bin/bash

mkdir certs
export ROX_MTLS_CERT_FILE=$PWD/certs/cert.pem
export ROX_SENSOR_ENDPOINT=sensor.default.svc:443
export ROX_MTLS_CA_FILE=$PWD/certs/ca.pem
kubectl get secret tls-cert-admission-control -o json | jq -r '.data["cert.pem"]' | base64 -d > certs/cert.pem
kubectl get secret additional-ca -o json | jq -r '.data["ca.pem"]' | base64 -d > certs/ca.pem

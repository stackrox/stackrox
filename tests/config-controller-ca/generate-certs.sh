#!/bin/bash

# Generates CA and server certificates for config-controller additional CA tests.
# The server cert has SANs for the qa-config-controller-ca namespace.

set -euo pipefail

cd "$(dirname "$0")"

NAMESPACE="qa-config-controller-ca"
SERVICE_NAME="nginx-proxy"

echo "=== Generating Root CA ==="
openssl genrsa -out root-ca.key 3072
openssl req -x509 -nodes -sha256 -new -key root-ca.key -out root-ca.crt -days $((50*365)) \
    -subj "/CN=Config Controller Test CA" \
    -addext "keyUsage = critical, keyCertSign" \
    -addext "basicConstraints = critical, CA:TRUE, pathlen:0" \
    -addext "subjectKeyIdentifier = hash"

echo "=== Generating Server Certificate ==="
openssl genrsa -out server.key 2048
openssl req -sha256 -new -key server.key -out server.csr \
    -subj "/CN=${SERVICE_NAME}.${NAMESPACE}/O=StackRox Tests/OU=Config Controller QA" \
    -reqexts SAN -config <(cat <<EOF
[dn]
CN=localhost
[req]
distinguished_name = dn
[SAN]
subjectAltName=DNS:${SERVICE_NAME}.${NAMESPACE},DNS:*.${NAMESPACE},DNS:${SERVICE_NAME}.${NAMESPACE}.svc,DNS:${SERVICE_NAME}.${NAMESPACE}.svc.cluster.local,IP:127.0.0.1
EOF
)

openssl x509 -req -sha256 -in server.csr -out server.crt -days $((50*365)) \
    -CAkey root-ca.key -CA root-ca.crt -CAcreateserial \
    -extfile <(cat <<EOF
subjectAltName = DNS:${SERVICE_NAME}.${NAMESPACE},DNS:*.${NAMESPACE},DNS:${SERVICE_NAME}.${NAMESPACE}.svc,DNS:${SERVICE_NAME}.${NAMESPACE}.svc.cluster.local,IP:127.0.0.1
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
basicConstraints = CA:FALSE
authorityKeyIdentifier = keyid:always
subjectKeyIdentifier = none
EOF
)

echo "=== Cleaning up temporary files ==="
rm -f server.csr root-ca.key root-ca.srl

echo "=== Generated files ==="
ls -la *.crt *.key

echo ""
echo "=== Root CA Certificate ==="
openssl x509 -in root-ca.crt -noout -subject -issuer

echo ""
echo "=== Server Certificate ==="
openssl x509 -in server.crt -noout -subject -issuer
echo "SANs:"
openssl x509 -in server.crt -noout -ext subjectAltName

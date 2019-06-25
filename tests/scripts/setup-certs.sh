#!/usr/bin/env bash

set -euo pipefail

target_dir="$1"
cn="$2"

[[ -n "$cn" ]] || { echo >&2 "No CN specified!" ; exit 1 ; }

ca_name="${3:-Test CA}"

[[ -d "$target_dir" ]] || mkdir "$target_dir"
chmod 0700 "$target_dir"

cd "$target_dir"

# Extensions for intermediate CA
intermediate_ca_exts='
basicConstraints = critical, CA:true, pathlen:0
keyUsage = critical, digitalSignature, cRLSign, keyCertSign
'

# Root CA
openssl req -nodes -new -x509 -keyout ca.key -out ca.crt -subj "/CN=Root ${ca_name}"

# Intermediate CA
openssl genrsa -out intermediate.key 2048
openssl req -new -key intermediate.key -subj "/CN=Intermediate ${ca_name}" \
    | openssl x509 -extfile <(echo "$intermediate_ca_exts") -req -CA ca.crt -CAkey ca.key -CAcreateserial -out intermediate.crt

# Leaf cert
openssl genrsa -out leaf.key 2048
openssl req -new -key leaf.key -subj "/CN=${cn}" \
    | openssl x509 -req -CA intermediate.crt -CAkey intermediate.key -CAcreateserial -out leaf.crt

cat leaf.crt intermediate.crt ca.crt >tls.crt
cp leaf.key tls.key

openssl pkcs12 -export -inkey tls.key -in tls.crt -out keystore.p12 -passout pass:

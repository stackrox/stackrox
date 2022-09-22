#!/usr/bin/env bash
set -eo pipefail

root_ca_exts="
  [req]
  distinguished_name=dn
  x509_extensions=ext
  [ dn ]
  [ ext ]
  basicConstraints=CA:TRUE,pathlen:0
"

openssl req -config <(echo "${root_ca_exts}") -nodes -new -x509 -newkey 4096 -keyout ca-key.pem -out ca.pem -days 1825 -subj "/CN=StackRox Certificate Authority"


leaf_ca_exts="subjectAltName=DNS:central.stackrox"
openssl genrsa -out leaf-key.pem 4096
openssl req -new -key leaf-key.pem -subj "/CN=StackRox Central" | \
     openssl x509 -sha256 -extfile <(echo "$leaf_ca_exts") -req -CAcreateserial -CA ca.pem -CAkey ca-key.pem -out leaf-cert.pem -days 1825

rm ca.srl

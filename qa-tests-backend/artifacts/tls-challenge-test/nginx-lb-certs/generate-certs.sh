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

openssl req -config <(echo "${root_ca_exts}") -nodes -new -x509 -newkey 2048 -keyout ca-key.pem -out ca.pem -days 1825 -subj "/CN=LoadBalancer Certificate Authority"

leaf_ca_exts="subjectAltName=DNS:nginx-loadbalancer.qa-tls-challenge"
openssl genrsa -out leaf-key.pem 2048
openssl req -new -key leaf-key.pem -subj "/CN=nginx LoadBalancer" | \
     openssl x509 -extfile <(echo "${leaf_ca_exts}") -req -CAcreateserial -CA ca.pem -CAkey ca-key.pem -out leaf-cert.pem -days 1825

openssl x509 -outform der -in leaf-key.pem

rm ca.srl

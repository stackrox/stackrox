#!/usr/bin/env bash
set -eou pipefail

kubectl -n stackrox get secret stackrox -o yaml > stackrox-secret.yml
sed 's|stackrox|qa|' stackrox-secret.yml > qa-secret.yml
kubectl create -f qa-secret.yml

rm stackrox-secret.yml
rm qa-secret.yml

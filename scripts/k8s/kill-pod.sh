#! /bin/bash

NAME=$1
NAMESPACE=${2:-stackrox}

kubectl -n "$NAMESPACE" delete po $(kubectl -n "$NAMESPACE" get po --selector app="$NAME" -o jsonpath='{.items[].metadata.name}') --grace-period=0

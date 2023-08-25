#! /bin/bash

NAME=$1

kubectl -n stackrox delete po $(kubectl -n stackrox get po --selector app="$NAME" -o jsonpath='{.items[].metadata.name}') --grace-period=0

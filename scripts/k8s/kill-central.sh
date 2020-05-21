#! /bin/bash

kubectl -n stackrox delete po $(kubectl -n stackrox get po --selector app=central -o jsonpath='{.items[].metadata.name}') --grace-period=0

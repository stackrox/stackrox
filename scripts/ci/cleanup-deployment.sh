#! /bin/bash

kubectl -n stackrox get cm,deploy,ds,networkpolicy,pv,pvc,secret,svc,serviceaccount -o name | xargs kubectl -n stackrox delete

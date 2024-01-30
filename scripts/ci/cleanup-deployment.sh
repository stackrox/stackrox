#!/usr/bin/env bash

namespace=${1:-stackrox}

kubectl -n "${namespace}" get cm,deploy,ds,networkpolicy,pv,pvc,secret,svc,serviceaccount -o name | xargs kubectl -n "${namespace}" delete --wait

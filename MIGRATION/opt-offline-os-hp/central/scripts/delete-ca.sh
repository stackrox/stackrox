#!/usr/bin/env bash

KUBE_COMMAND=${KUBE_COMMAND:-oc}

${KUBE_COMMAND} delete -n "stackrox" secret/additional-ca

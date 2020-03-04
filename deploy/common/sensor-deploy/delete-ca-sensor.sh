#!/usr/bin/env bash

KUBE_COMMAND=${KUBE_COMMAND:-kubectl}

${KUBE_COMMAND} delete -n "stackrox" secret/additional-ca-sensor

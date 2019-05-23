#!/usr/bin/env bash

KUBE_COMMAND=${KUBE_COMMAND:-{{.K8sConfig.Command}}}

{{.K8sConfig.Command}} delete -n "stackrox" secret/additional-ca

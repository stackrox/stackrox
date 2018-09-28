#!/usr/bin/env bash

kubectl delete -n "{{.K8sConfig.Namespace}}" secret/additional-ca

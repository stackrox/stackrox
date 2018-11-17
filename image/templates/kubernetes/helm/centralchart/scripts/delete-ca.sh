#!/usr/bin/env bash

{{.K8sConfig.Command}} delete -n "{{.K8sConfig.Namespace}}" secret/additional-ca

#!/usr/bin/env bash

{{.K8sConfig.Command}} delete -n "stackrox" secret/additional-ca

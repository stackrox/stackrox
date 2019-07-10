#!/usr/bin/env bash

{{  $kubeCmd := "" -}}
{{- $secretName := "additional-ca" -}}
{{- if .K8sConfig -}}
{{- $kubeCmd = .K8sConfig.Command -}}
{{- else -}}
{{- $kubeCmd = .K8sCommand -}}
{{- $secretName = "additional-ca-sensor" -}}
{{- end -}}
{{- if not $kubeCmd -}}
{{- $kubeCmd = "kubectl" -}}
{{- end -}}

KUBE_COMMAND=${KUBE_COMMAND:-{{$kubeCmd}}}

${KUBE_COMMAND} delete -n "stackrox" secret/{{$secretName}}

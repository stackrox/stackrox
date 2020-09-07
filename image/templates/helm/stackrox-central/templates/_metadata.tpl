{{/*
  srox.defaultLabels $

  Returns the default labels to be used for every object created by this chart.
   */}}
{{- define "srox.defaultLabels" -}}
app.kubernetes.io/name: stackrox
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/part-of: stackrox-central-services
{{- $component := regexReplaceAll "^.*/\\d{2}-([a-z]+)-\\d{2}-[^/]+\\.yaml" .Template.Name "${1}" -}}
{{- if not (contains "/" $component) }}
app.kubernetes.io/component: {{ $component | quote }}
{{- end }}
{{- end }}

{{/*
  srox.defaultAnnotations $

  Returns the default annotations to be used for every object created by this chart.
   */}}
{{- define "srox.defaultAnnotations" -}}
meta.helm.sh/release-namespace: {{ .Release.Namespace }}
meta.helm.sh/release-name: {{ .Release.Name }}
owner: stackrox
email: support@stackrox.com
{{- end }}

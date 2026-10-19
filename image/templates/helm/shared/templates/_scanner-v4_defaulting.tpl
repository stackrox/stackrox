{{/*
  srox.scannerV4Defaulting <Helm .> <Helm .Release> <Scanner V4 configuration> <Stackrox Helm ConfigMap content>

  Encapsulates the Scanner V4 defaulting logic.
*/}}

{{- define "srox.scannerV4Defaulting" -}}

{{- $ := index . 0 -}}
{{- $helmRelease := index . 1 -}}
{{- $scannerV4 := index . 2 -}}
{{- $stackroxHelm := index . 3 -}}

{{- if kindIs "invalid" $scannerV4.disable -}}
  {{/* Scanner V4 neither explicitly enabled or disabled. Default to enabled. */}}
  {{- $_ := set $scannerV4 "disable" false -}}
{{- end -}}
{{- end -}}

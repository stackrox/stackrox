{{/*
  srox.scannerV4CentralDefaulting <Helm .Release> <Scanner V4 configuration>

  Encapsulates the Scanner V4 defaulting logic for central-services.
*/}}

{{- define "srox.scannerV4CentralDefaulting" -}}

{{- $helmRelease := index . 0 -}}
{{- $scannerV4 := index . 1 -}}

{{- if kindIs "invalid" $scannerV4.disable -}}

  {{/* Default to not-installed (i.e. upgrades). */}}
  {{- $_ := set $scannerV4 "disable" true -}}

  {{/* Currently the automatic enabling of Scanner V4 only kicks in for new installations, not for upgrades. */}}
  {{- if $helmRelease.IsInstall -}}
    {{- $_ := set $scannerV4 "disable" false -}}
  {{- end -}}

{{- end -}}

{{- end -}}

{{/*
  srox.scannerDefaulting <Helm .Release> <Scanner configuration>

  Encapsulates the Scanner defaulting logic.
  
  Can be removed later, together with StackRox Scanner.
*/}}

{{- define "srox.scannerDefaulting" -}}

{{- $helmRelease := index . 0 -}}
{{- $scanner := index . 1 -}}

{{- if kindIs "invalid" $scanner.disable -}}

  {{/* Default to not-installed (i.e. upgrades). */}}
  {{- $_ := set $scanner "disable" true -}}

  {{/* Currently the automatic enabling of Scanner only kicks in for new installations, not for upgrades. */}}
  {{- if $helmRelease.IsInstall -}}
    {{- $_ := set $scanner "disable" false -}}
  {{- end -}}

{{- end -}}

{{- end -}}

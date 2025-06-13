{{/*
  srox.scannerDefaulting $ <Scanner configuration>

  Encapsulates the Scanner defaulting logic.

  Can be removed later, together with StackRox Scanner.
*/}}

{{- define "srox.scannerDefaulting" -}}

{{- $ := index . 0 -}}
{{- $scanner := index . 1 -}}

{{- if kindIs "invalid" $scanner.disable -}}

  {{/* Default to not-installed (i.e. upgrades). */}}
  {{- $_ := set $scanner "disable" true -}}

  {{/* Currently the automatic enabling of Scanner only kicks in for new installations, not for upgrades. */}}
  {{- if $.Release.IsInstall -}}
    {{- $_ := set $scanner "disable" false -}}
  {{- end -}}

  {{/* If this is an upgrade and Scanner is installed, it should stay installed. */}}
  {{- if $scanner.disable -}}
    {{ $scannerDeployment := dict }}
    {{ include "srox.safeLookup" (list $ $scannerDeployment "apps/v1" "Deployment" $.Release.Namespace "scanner") }}
    {{ if $scannerDeployment.result }}
      {{ include "srox.note" (list $ "Detected existing scanner installation, keeping Scanner installed.") }}
      {{- $_ := set $scanner "disable" false -}}
    {{ end }}
  {{- end -}}

{{- end -}}

{{- end -}}

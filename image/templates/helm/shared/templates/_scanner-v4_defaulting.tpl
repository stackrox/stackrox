{{/*
  srox.scannerV4Defaulting $ <Scanner V4 configuration>

  Encapsulates the Scanner V4 defaulting logic.
*/}}

{{- define "srox.scannerV4Defaulting" -}}

{{- $ := index . 0 -}}
{{- $scannerV4 := index . 1 -}}

{{- if kindIs "invalid" $scannerV4.disable -}}

  {{/* Default to not-installed (i.e. upgrades). */}}
  {{- $_ := set $scannerV4 "disable" true -}}

  {{/* Currently the automatic enabling of Scanner V4 only kicks in for new installations, not for upgrades. */}}
  {{- if $.Release.IsInstall -}}
    {{- $_ := set $scannerV4 "disable" false -}}
  {{- end -}}
  
  {{/* If this is an upgrade and Scanner V4 Indexer is installed, it should stay installed. */}}
  {{- if $scannerV4.disable -}}
    {{ $indexerDeployment := dict }}
    {{ include "srox.safeLookup" (list $ $indexerDeployment "apps/v1" "Deployment" $.Release.Namespace "scanner-v4-indexer") }}
    {{ if $indexerDeployment.result }}
      {{ include "srox.note" (list $ "Detected existing scanner-v4-indexer installation, keeping Scanner V4 installed.") }}
      {{- $_ := set $scannerV4 "disable" false -}}
    {{ end }}
  {{- end -}}

{{- end -}}

{{- end -}}

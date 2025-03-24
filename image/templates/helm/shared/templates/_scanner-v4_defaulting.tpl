{{/*
  srox.scannerV4CentralDefaulting <Helm .Release> <Scanner V4 configuration> <Scanner V2 configuration>

  Encapsulates the Scanner V4 defaulting logic for central-services.
*/}}

{{- define "srox.scannerV4CentralDefaulting" -}}

{{- $helmRelease := index . 0 -}}
{{- $scannerV4 := index . 1 -}}
{{- $scanner := index . 2 -}}

{{- if kindIs "invalid" $scannerV4.disable -}}

  {{/* Default to not-installed. */}}
  {{- $_ := set $scannerV4 "disable" true -}}

  {{/* Currently the automatic enabling of Scanner V4 only kicks in for new installations, not for upgrades. */}}
  {{- if $helmRelease.IsInstall -}}
    {{/*
        Here we are trying to not cause any surprises for the user: If he disabled scanner V2 explicitly,
        without also doing the same for scanner V4, it could be that he did not take scanner V4's existence
        into account, hence we are interpreting this as "the user does not want scanner(s) at all" and we
        skip the enabling of scanner V4.
    */}}
    {{- if not $scanner.disable -}}
      {{- $_ := set $scannerV4 "disable" false -}}
    {{- end -}}
  {{- end -}}

{{- end -}}

{{- end -}}

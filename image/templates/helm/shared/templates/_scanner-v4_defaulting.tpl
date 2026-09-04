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
  {{/* Scanner V4 neither explicitly enabled or disabled, apply defaulting logic. */}}
  {{/* By default Scanner V4 will be installed, for both fresh installs and upgrades.
       Since the legacy StackRox Scanner is retired, the previous pre-4.8 upgrade
       exception (which left Scanner V4 disabled) has been removed so that upgrades that
       relied on defaulting migrate to Scanner V4. An explicit scannerV4.disable: true is
       still honored. */}}
  {{- $_ := set $scannerV4 "disable" false -}}
{{- end -}}
{{- end -}}

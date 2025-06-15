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
  {{/* By default Scanner V4 will be installed. */}}
  {{- $_ := set $scannerV4 "disable" false -}}
  {{/* Currently there is one exception: when upgrading from a pre-4.8 version, which did not
       install Scanner V4 by default. */}}
  {{- $installVersionUnknown := kindIs "invalid" $stackroxHelm.installXYVersion -}}
  {{- $upgradingFromPre4_8 := or $installVersionUnknown (semverCompare "< 4.8" $stackroxHelm.installXYVersion) -}}
  {{- if and $helmRelease.IsUpgrade $upgradingFromPre4_8 -}}
    {{- include "srox.note" (list $ (printf "Scanner V4 disabled due to upgrade from version %s.x" $stackroxHelm.installXYVersion)) -}}
    {{- $_ := set $scannerV4 "disable" true -}}
  {{- end -}}
{{- end -}}
{{- end -}}

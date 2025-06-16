{{/*
  srox.scannerDefaulting <Helm .> <Helm .Release> <Scanner configuration> <Stackrox Helm ConfigMap content>

  Encapsulates the Scanner defaulting logic.

  Can be removed later, together with StackRox Scanner.
*/}}

{{- define "srox.scannerDefaulting" -}}

{{- $ := index . 0 -}}
{{- $helmRelease := index . 1 -}}
{{- $scanner := index . 2 -}}
{{- $stackroxHelm := index . 3 -}}

{{- if kindIs "invalid" $scanner.disable -}}
  {{/* Scanner neither explicitly enabled or disabled, apply defaulting logic. */}}
  {{/* By default Scanner will be installed. */}}
  {{- $_ := set $scanner "disable" false -}}
  {{/* Currently there is one exception: when upgrading from a pre-4.8 version, which did not
       install Scanner by default. */}}
  {{- $installVersionUnknown := kindIs "invalid" $stackroxHelm.installXYVersion -}}
  {{- $upgradingFromPre4_8 := or $installVersionUnknown (semverCompare "< 4.8" $stackroxHelm.installXYVersion) -}}
  {{- if and $helmRelease.IsUpgrade $upgradingFromPre4_8 -}}
    {{- include "srox.note" (list $ "StackRox Scanner disabled by default: this deployment was initially installed before version 4.8.") -}}
    {{- $_ := set $scanner "disable" true -}}
  {{- end -}}
{{- end -}}
{{- end -}}

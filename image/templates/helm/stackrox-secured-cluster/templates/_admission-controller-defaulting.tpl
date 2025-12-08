{{/*
  srox.admissionControllerDefaulting <Helm .> <Helm .Release> <Admission Control configuration> <Stackrox Helm ConfigMap content>

  Encapsulates Admission Controller defaulting logic.
*/}}

{{- define "srox.admissionControllerDefaulting" -}}

{{- $ := index . 0 -}}
{{- $helmRelease := index . 1 -}}
{{- $admissionControl := index . 2 -}}
{{- $stackroxHelm := index . 3 -}}

{{- $installVersionUnknown := kindIs "invalid" $stackroxHelm.installXYVersion -}}
{{- $upgradingFromPre4_9 := or $installVersionUnknown (semverCompare "< 4.9" $stackroxHelm.installXYVersion) -}}

{{/* Warn if old enforceOn* options are used. We don't unset them here, because they are used for the defaulting logic during upgrades. */}}
{{- if or $admissionControl.dynamic.enforceOnCreates $admissionControl.dynamic.enforceOnUpdates -}}
  {{- include "srox.warn" (list $ "The fields 'enforceOnCreates' and 'enforceOnUpdates' are deprecated and will be ignored. Use the new 'enforce' setting instead.") -}}
{{- end -}}

{{- if kindIs "invalid" $admissionControl.enforce -}}
  {{/* New-style enforcement neither explicitly enabled or disabled => apply defaulting logic. */}}

  {{/* By default enforcement is enabled. */}}
  {{- $_ := set $admissionControl "enforce" true -}}

  {{/* Implement defaulting for upgrades from previous version. */}}
  {{- if and $helmRelease.IsUpgrade $upgradingFromPre4_9 -}}
    {{/* In this upgrade scenario we will defeault to enforce=false, but will upgrade to enforce=true if at least one of the old enforceOn* settings is enabled. */}}
    {{- $_ := set $admissionControl "enforce" false -}}

    {{- if $admissionControl.dynamic.enforceOnCreates -}}
      {{- $note := "Detected upgrade from pre-4.9: Admission Controller enforcement will be generally turned on, because enforceOnCreates is enabled." -}}
      {{- include "srox.warn" (list $ $note) -}}
      {{- $_ := set $admissionControl "enforce" true -}}
    {{- end -}}
    {{- if $admissionControl.dynamic.enforceOnUpdates -}}
      {{- $note := "Detected upgrade from pre-4.9: Admission Controller enforcement will be generally turned on, because enforceOnUpdates is enabled." -}}
      {{- include "srox.warn" (list $ $note) -}}
      {{- $_ := set $admissionControl "enforce" true -}}
    {{- end -}}

  {{- end -}}
{{- end -}}

{{- if not $admissionControl.enforce -}}
  {{- $note := "Admission Controller enforcement will be completely disabled, this is a bad idea. Please consult the documentation for more information." -}}
  {{- include "srox.warn" (list $ $note) -}}
{{- end -}}

{{/* Propagate new high-level field to internally used low-level fields. */}}
{{- $_ := set $admissionControl.dynamic "enforceOnCreates" $admissionControl.enforce -}}
{{- $_ := set $admissionControl.dynamic "enforceOnUpdates" $admissionControl.enforce -}}

{{- end -}}

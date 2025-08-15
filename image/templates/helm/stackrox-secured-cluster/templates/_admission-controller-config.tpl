{{- define "srox.protectAdmissionControllerConfig" -}}
{{- $ := . -}}

{{- $formatMsg := "It is not supported anymore to specify 'admissionControl.%s'. This setting will be ignored. The effective value is '%v'." -}}

{{/* listenOn* fields. */}}
{{- if not (kindIs "invalid" $._rox.admissionControl.listenOnCreates) -}}
    {{- include "srox.warn" (list $ (printf $formatMsg "listenOnCreates" $._rox._defaults.admissionControl.listenOnCreates)) -}}
    {{- $_ := unset $._rox.admissionControl "listenOnCreates" -}}
{{- end -}}
{{- if not (kindIs "invalid" $._rox.admissionControl.listenOnUpdates) -}}
    {{- include "srox.warn" (list $ (printf $formatMsg "listenOnUpdates" $._rox._defaults.admissionControl.listenOnUpdates)) -}}
    {{- $_ := unset $._rox.admissionControl "listenOnUpdates" -}}
{{- end -}}
{{- if not (kindIs "invalid" $._rox.admissionControl.listenOnEvents) -}}
    {{- include "srox.warn" (list $ (printf $formatMsg "listenOnEvents" $._rox._defaults.admissionControl.listenOnEvents)) -}}
    {{- $_ := unset $._rox.admissionControl "listenOnEvents" -}}
{{- end -}}

{{/* scanInline field. */}}
{{- if not (kindIs "invalid" $._rox.admissionControl.dynamic.scanInline) -}}
    {{- include "srox.warn" (list $ (printf $formatMsg "dynamic.scanInline" $._rox._defaults.admissionControl.dynamic.scanInline)) -}}
    {{- $_ := unset $._rox.admissionControl.dynamic "scanInline" -}}
{{- end -}}

{{/* timeout field. */}}
{{- if not (kindIs "invalid" $._rox.admissionControl.dynamic.timeout) -}}
    {{- include "srox.warn" (list $ (printf $formatMsg "dynamic.timeout" $._rox._defaults.admissionControl.dynamic.timeout)) -}}
    {{- $_ := unset $._rox.admissionControl.dynamic "timeout" -}}
{{- end -}}

{{- end -}}

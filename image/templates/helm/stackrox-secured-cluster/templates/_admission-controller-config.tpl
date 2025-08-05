{{- define "srox.protectAdmissionControllerConfig" -}}
{{- $ := . -}}

{{- $formatMsg := "It is not supported anymore to specify 'admissionControl.%s'. This setting will be ignored. The effective value is 'true'." -}}

{{/* listenOn* fields. */}}
{{- if not (kindIs "invalid" $._rox.admissionControl.listenOnCreates) -}}
    {{- include "srox.warn" (list $ (printf $formatMsg "listenOnCreates")) -}}
    {{- $_ := unset $._rox.admissionControl "listenOnCreates" -}}
{{- end -}}
{{- if not (kindIs "invalid" $._rox.admissionControl.listenOnUpdates) -}}
    {{- include "srox.warn" (list $ (printf $formatMsg "listenOnUpdates")) -}}
    {{- $_ := unset $._rox.admissionControl "listenOnUpdates" -}}
{{- end -}}
{{- if not (kindIs "invalid" $._rox.admissionControl.listenOnEvents) -}}
    {{- include "srox.warn" (list $ (printf $formatMsg "listenOnEvents")) -}}
    {{- $_ := unset $._rox.admissionControl "listenOnEvents" -}}
{{- end -}}

{{/* scanInline field. */}}
{{- if not (kindIs "invalid" $._rox.admissionControl.dynamic.scanInline) -}}
    {{- include "srox.warn" (list $ (printf $formatMsg "dynamic.scanInline")) -}}
    {{- $_ := unset $._rox.admissionControl.dynamic "scanInline" -}}
{{- end -}}

{{- end -}}

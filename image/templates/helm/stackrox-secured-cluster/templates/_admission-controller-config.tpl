{{- define "srox.protectAdmissionControllerConfig" -}}
{{- $ := . -}}

{{- $formatMsg := "It is not supported anymore to specify 'admissionControl.%s'. This setting will be ignored." -}}

{{- if not (kindIs "invalid" $._rox.admissionControl.listenOnCreates) -}}
    {{- include "srox.warn" (list $ (printf $formatMsg "listenOnCreates")) -}}
{{- end -}}
{{- if not (kindIs "invalid" $._rox.admissionControl.listenOnUpdates) -}}
    {{- include "srox.warn" (list $ (printf $formatMsg "listenOnUpdates")) -}}
{{- end -}}
{{- if not (kindIs "invalid" $._rox.admissionControl.listenOnEvents) -}}
    {{- include "srox.warn" (list $ (printf $formatMsg "listenOnEvents")) -}}
{{- end -}}

{{- end -}}

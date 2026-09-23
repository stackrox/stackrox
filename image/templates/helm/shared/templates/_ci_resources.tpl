{{/*
  CI opts into this policy through customize.annotations so operator-managed and
  Helm deployments share it without changing production resource defaults.
  Apply after defaulting: omitted requests/limits would otherwise be restored.
*/}}
{{- define "srox.ciResources" -}}
{{- $ := index . 0 -}}
{{- $resources := index . 1 -}}
{{- $path := index . 2 -}}
{{- $annotations := $._rox.customize.annotations | default dict -}}
{{- if eq (index $annotations "ci.stackrox.io/resource-policy" | default "") "requests" -}}
  {{- $resources = deepCopy $resources -}}
  {{- $requests := $resources.requests | default dict -}}
  {{- $limits := $resources.limits | default dict -}}
  {{- $memory := $requests.memory | default $limits.memory -}}
  {{- if eq $path "central.resources" -}}
    {{- $memory = "1Gi" -}}
  {{- end -}}
  {{- if $memory -}}
    {{- $_ := set $requests "memory" $memory -}}
    {{- $_ := set $limits "memory" $memory -}}
  {{- end -}}
  {{- if and (not (hasKey $requests "cpu")) (hasKey $limits "cpu") -}}
    {{- $_ := set $requests "cpu" $limits.cpu -}}
  {{- end -}}
  {{- $_ := unset $limits "cpu" -}}
  {{- $_ := set $resources "requests" $requests -}}
  {{- $_ := set $resources "limits" $limits -}}
{{- end -}}
{{- toYaml $resources -}}
{{- end -}}

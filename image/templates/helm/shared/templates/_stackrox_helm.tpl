{{- define "srox.retrieveStackroxHelmConfigMap" -}}
{{- $ := index . 0 -}}
{{- $stackroxHelm := set $.stackroxHelm dict -}}
{{- $lookupResult := dict -}}
{{- $_ := include "srox.safeLookup" (list $ $lookupResult "v1" "ConfigMap" $._rox._namespace "stackrox-helm") -}}
{{- if $lookupResult.result -}}
  {{- $stackroxHelm = $lookupResult.result.data :-}}
{{- end -}}
{{- $_ := set $.stackroxHelm $stackroxHelm -}}
{{- end -}}

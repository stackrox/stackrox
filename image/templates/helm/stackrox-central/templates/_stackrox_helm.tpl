{{- define "srox.retrieveStackroxCentralHelmConfigMap" -}}
{{- $ := index . 0 -}}
{{- $stackroxHelm := dict -}}
{{- $lookupResult := dict -}}
{{- $_ := include "srox.safeLookup" (list $ $lookupResult "v1" "ConfigMap" $.Release.Namespace "stackrox-central-helm") -}}
{{- if $lookupResult.result -}}
  {{- $stackroxHelm = $lookupResult.result.data -}}
{{- end -}}
{{- $_ := set $ "stackroxHelm" $stackroxHelm -}}
{{- end -}}

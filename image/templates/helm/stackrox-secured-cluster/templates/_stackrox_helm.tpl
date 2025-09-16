{{- define "srox.retrieveStackroxSecuredClusterHelmConfigMap" -}}
{{- $ := index . 0 -}}
{{- $stackroxHelm := dict -}}
{{- $lookupResult := dict -}}
{{- $_ := include "srox.safeLookup" (list $ $lookupResult "v1" "ConfigMap" $.Release.Namespace "stackrox-secured-cluster-helm") -}}
{{- if $lookupResult.result -}}
  {{- $stackroxHelm = $lookupResult.result.data -}}
{{- end -}}
{{- $_ := set $ "stackroxHelm" $stackroxHelm -}}
{{- end -}}

{{/*
    srox.autoSensePodSecurityPolicies $
  */}}

{{ define "srox.autoSensePodSecurityPolicies" }}

{{ $ := index . 0 }}
{{ $system := $._rox.system }}

{{ if kindIs "invalid" $system.enablePodSecurityPolicies }}
  {{ $_ := set $system "enablePodSecurityPolicies" (has "policy/v1beta1" $._rox._apiServer.apiResources) }}
  {{ if $system.enablePodSecurityPolicies }}
    {{ include "srox.note" (list $ (printf "PodSecurityPolicies are enabled, since your environment supports them according to API server properties.")) }}
  {{ else }}
    {{ include "srox.note" (list $ (printf "PodSecurityPolicies are disabled, since your environment does not support them according to API server properties.")) }}
  {{ end }}
{{ end }}

{{ end }}

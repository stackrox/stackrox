{{/*
    srox.autoSensePodSecurityPolicies $

    DEPRECATED: PodSecurityPolicy support is deprecated and will be removed in a
    future release. Kubernetes removed PodSecurityPolicy in v1.25. If you are
    running on Kubernetes >= 1.25, PSPs are already inactive. Consider migrating
    to Pod Security Admission or other policy engines.
  */}}

{{ define "srox.autoSensePodSecurityPolicies" }}

{{ $ := index . 0 }}
{{ $system := $._rox.system }}

{{ if kindIs "invalid" $system.enablePodSecurityPolicies }}
  {{ $_ := set $system "enablePodSecurityPolicies" (has "policy/v1beta1" $._rox._apiServer.apiResources) }}
  {{ if $system.enablePodSecurityPolicies }}
    {{ include "srox.note" (list $ (printf "PodSecurityPolicies are enabled, since your environment supports them according to API server properties.")) }}
    {{ include "srox.warn" (list $ "PodSecurityPolicy support is deprecated and will be removed in a future release. Kubernetes removed PodSecurityPolicy in v1.25. Consider disabling PSPs by setting system.enablePodSecurityPolicies=false.") }}
  {{ else }}
    {{ include "srox.note" (list $ (printf "PodSecurityPolicies are disabled, since your environment does not support them according to API server properties.")) }}
  {{ end }}
{{ else if $system.enablePodSecurityPolicies }}
  {{ include "srox.warn" (list $ "PodSecurityPolicy support is deprecated and will be removed in a future release. Kubernetes removed PodSecurityPolicy in v1.25. Please plan to set system.enablePodSecurityPolicies=false and migrate to Pod Security Admission or another policy engine.") }}
{{ end }}

{{ end }}

{{/*
    srox.autoSenseOpenshiftVersion $

    This function detects the OpenShift version automatically based on the cluster the Helm chart is installed onto.
    It writes the result to ._rox.env.openshift as an integer.
    Possible results are:
     - 3 (OpenShift 3)
     - 4 (OpenShift 4)
     - 0 (Non-Openshift cluster)

    If "true" is passed for $._rox.env.openshift the OpenShift version is detected based on the Kubernetes cluster version.
    If the Kubernetes version is not available (i.e. when using Helm template) auto-sensing falls back on OpenShift 3 to be
    backward compatible.
  */}}

{{- define "srox.autoSenseOpenshiftVersion" -}}

{{- $ := index . 0 -}}
{{- $env := $._rox.env -}}

{{/* Either OpenShift version (3, 4) or 0 to indicate "non-OpenShift". */}}
{{- $autoSensedOpenShift := 0 -}}
{{- $normalizedUserSpecifiedOpenShift := $env.openshift -}}

{{/* Normalize user setting.*/}}
{{- if kindIs "string" $normalizedUserSpecifiedOpenShift -}}
  {{/* User set something like env.openshift="4". */}}
  {{- $normalizedUserSpecifiedOpenShift = int $normalizedUserSpecifiedOpenShift -}}
{{- end -}}
{{- if kindIs "float64" $normalizedUserSpecifiedOpenShift -}}
  {{/* YAML numeric values are often parsed as float64. */}}
  {{- $normalizedUserSpecifiedOpenShift = int $normalizedUserSpecifiedOpenShift -}}
{{- end -}}

{{- if kindIs "invalid" $normalizedUserSpecifiedOpenShift -}}
  {{/* env.openshift is unset. */}}
{{- else if and (kindIs "int" $normalizedUserSpecifiedOpenShift) (or (eq $normalizedUserSpecifiedOpenShift 3) (eq $normalizedUserSpecifiedOpenShift 4)) -}}
  {{/* env.openshift=3,4. */}}
{{- else if kindIs "bool" $normalizedUserSpecifiedOpenShift -}}
  {{/* env.openshift=true,false. */}}
{{- else -}}
  {{/* This indicates that the original user setting $env.openshift is bogus -- normalization didn't help us. */}}
  {{- include "srox.fail" (printf "Invalid user setting for env.openshift=%v: Either un-set to rely on auto-sensing or set to an OpenShift version (3 or 4) or to a boolean" $normalizedUserSpecifiedOpenShift) -}}
{{- end -}}

{{/* $normalizedUserSpecifiedOpenShift is now nil (unset), 3, 4, false or true. */}}

{{/* Auto-sense OpenShift version in any case to warn about potential user errors, keeping the logic backwards compatible. */}}
{{- $hasConfigOpenShiftAPI := has "config.openshift.io/v1" $._rox._apiServer.apiResources -}}
{{- $hasProjectOpenShiftAPI := has "project.openshift.io/v1" $._rox._apiServer.apiResources -}}
{{- if $hasConfigOpenShiftAPI -}}
  {{/* This CRD API reliably indicates OpenShift 4. */}}
  {{- $autoSensedOpenShift = 4 -}}
{{- else if or (and (kindIs "bool" $normalizedUserSpecifiedOpenShift) $normalizedUserSpecifiedOpenShift) $hasProjectOpenShiftAPI -}}
  {{/* The API GroupVersion project.openshift.io/v1 contains the core OpenShift API 'Project' of
      compatibility level 1, which comes with the strongest stability guarantees among the OpenShift APIs.
      This API is available in OpenShift 3.x and 4.x, but unfortunately it's presence can fluctuate during
      OCP upgrades, hence checking for this API availability is not our first choice. */}}
  {{/* Parse and add KubeVersion as semver from built-in resources. This is necessary to compare valid integer numbers. */}}
  {{ $kubeVersion := semver $.Capabilities.KubeVersion.Version }}

  {{/* Default to OpenShift 3 if no openshift resources are available, i.e. in helm template commands */}}
  {{ if not $hasProjectOpenShiftAPI -}}
    {{- $autoSensedOpenShift = 3 }}
  {{ else if gt $kubeVersion.Minor 11 }}
    {{- $autoSensedOpenShift = 4 }}
  {{ else }}
    {{- $autoSensedOpenShift = 3 }}
  {{ end }}
{{- end -}}

{{/* $autoSensedOpenShift is now 0, 3 or 4. */}}

{{- if eq $autoSensedOpenShift 0 -}}
  {{- include "srox.note" (list $ "Based on API server properties, the target cluster is a non-OpenShift cluster.") -}}
{{- else -}}
  {{- include "srox.note" (list $ (printf "Based on API server properties, we have inferred that you are deploying into an OpenShift %d.x cluster." $autoSensedOpenShift)) -}}
{{- end -}}

{{- if kindIs "invalid" $normalizedUserSpecifiedOpenShift -}}
  {{/* env.openshift is entirely unset, simply use auto-sensing result. */}}
  {{- $_ := set $env "openshift" $autoSensedOpenShift -}}
{{- else if and (kindIs "bool" $normalizedUserSpecifiedOpenShift) $normalizedUserSpecifiedOpenShift -}}
  {{/* User just used env.openshift=true, which still requires us to auto-sense OpenShift 3 vs 4. */}}
  {{- if eq $autoSensedOpenShift 0 -}}
    {{/* Might indicate a configuration problem. */}}
    {{- include "srox.warn" (list $ "Detected user setting env.openshift=true, but auto-sensing of OpenShift version failed -- treating as OpenShift 3 for backwards compatibility reasons.") -}}
    {{- $_ := set $env "openshift" 3 -}}
  {{- else -}}
    {{- $_ := set $env "openshift" $autoSensedOpenShift -}}
  {{- end -}}
{{- else if and (kindIs "bool" $normalizedUserSpecifiedOpenShift) (not $normalizedUserSpecifiedOpenShift) -}}
  {{/* User set env.openshift=false. */}}
  {{- if gt $autoSensedOpenShift 0 -}}
    {{/* Might indicate a configuration problem. */}}
    {{- include "srox.warn" (list $ (printf "User setting env.openshift=false is in contrast to the result of the auto-sensing, which yielded an OpenShift %d.x target cluster -- will proceed with user setting" $autoSensedOpenShift)) -}}
  {{- end -}}
  {{- $_ := set $env "openshift" 0 -}}
{{- else -}}
  {{/* User set env.openshift=3 or 4. */}}
  {{- if not (eq $normalizedUserSpecifiedOpenShift $autoSensedOpenShift) -}}
    {{/* Might indicate a configuration problem. */}}
    {{- include "srox.warn" (list $ (printf "User setting env.openshift=%d is in contrast to the result of the auto-sensing, which yielded an OpenShift %d.x target cluster -- will proceed with user setting" $normalizedUserSpecifiedOpenShift $autoSensedOpenShift)) -}}
  {{- end -}}
  {{- $_ := set $env "openshift" $normalizedUserSpecifiedOpenShift -}}
{{- end -}}

{{- end -}}

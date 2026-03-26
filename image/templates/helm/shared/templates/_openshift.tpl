{{/*
    srox.autoSenseOpenshiftVersion $

    This function detects the OpenShift version automatically based on the cluster the Helm chart is installed onto.
    It writes the result to ._rox.env.openshift as an integer.
    Possible results are:
     - 4 (OpenShift 4)
     - 0 (Non-Openshift cluster)
  */}}

{{- define "srox.autoSenseOpenshiftVersion" -}}

{{- $ := index . 0 -}}
{{- $env := $._rox.env -}}

{{/* Either OpenShift version (currently only 4) or 0 to indicate "non-OpenShift". */}}
{{- $autoSensedOpenShift := 0 -}}
{{- $normalizedUserSpecifiedOpenShift := $env.openshift -}}

{{/* Normalize user setting. */}}
{{- $userSpecifiedOpenShiftAsInt := int $normalizedUserSpecifiedOpenShift -}}
{{- if kindIs "string" $normalizedUserSpecifiedOpenShift -}}
  {{- if eq (printf "%d" $userSpecifiedOpenShiftAsInt) $normalizedUserSpecifiedOpenShift -}}
    {{/* User set something like env.openshift="4". We allow this as well. */}}
    {{- $normalizedUserSpecifiedOpenShift = $userSpecifiedOpenShiftAsInt -}}
  {{- end -}}
{{- end -}}
{{- if kindIs "float64" $normalizedUserSpecifiedOpenShift -}}
  {{/* YAML numeric values are sometimes parsed as float64... */}}
  {{- $normalizedUserSpecifiedOpenShift = $userSpecifiedOpenShiftAsInt -}}
{{- end -}}
{{- if kindIs "int64" $normalizedUserSpecifiedOpenShift -}}
  {{/* ... and sometimes as int64 (e.g. when doing helm --set env.openshift="4"). */}}
  {{- $normalizedUserSpecifiedOpenShift = $userSpecifiedOpenShiftAsInt -}}
{{- end -}}

{{/* Verify that we have a sensible user setting. */}}
{{- if kindIs "invalid" $normalizedUserSpecifiedOpenShift -}}
  {{/* env.openshift is unset. */}}
{{- else if and (kindIs "int" $normalizedUserSpecifiedOpenShift) (eq $normalizedUserSpecifiedOpenShift 4) -}}
  {{/* env.openshift=4. */}}
{{- else if kindIs "bool" $normalizedUserSpecifiedOpenShift -}}
  {{/* env.openshift=true,false. */}}
{{- else -}}
  {{/* This indicates that the original user setting $env.openshift is bogus -- normalization didn't help us. */}}
  {{- $fmt := printf
        "Invalid user setting for env.openshift=%v (%T): Either un-set to rely on auto-sensing or set to an OpenShift major version (currently only 4) or to a boolean"
        $normalizedUserSpecifiedOpenShift $normalizedUserSpecifiedOpenShift -}}
  {{- include "srox.fail" $fmt -}}
{{- end -}}

{{/* $normalizedUserSpecifiedOpenShift is now nil (unset), 4 (int), or false/true (bool). */}}

{{/* Auto-sense OpenShift version in any case to warn about potential user errors. */}}
{{- if has "config.openshift.io/v1" $._rox._apiServer.apiResources -}}
  {{/* This CRD API reliably indicates OpenShift 4. */}}
  {{- $autoSensedOpenShift = 4 -}}
{{- end -}}

{{/* $autoSensedOpenShift is now 0 or 4. */}}

{{- if eq $autoSensedOpenShift 0 -}}
  {{- include "srox.note" (list $ "Based on API server properties, the target cluster is a non-OpenShift cluster.") -}}
{{- else -}}
  {{- include "srox.note" (list $ (printf "Based on API server properties, we have inferred that you are deploying into an OpenShift %d.x cluster." $autoSensedOpenShift)) -}}
{{- end -}}

{{/* Propagate the final decision on the OpenShift version to $env.openshift. */}}
{{- if kindIs "invalid" $normalizedUserSpecifiedOpenShift -}}
  {{/* env.openshift is entirely unset, simply use auto-sensing result. */}}
  {{- $_ := set $env "openshift" $autoSensedOpenShift -}}
{{- else if and (kindIs "bool" $normalizedUserSpecifiedOpenShift) $normalizedUserSpecifiedOpenShift -}}
  {{/* User just used env.openshift=true, which is treated as equivalent to env.openshift=4. */}}
  {{- if ne $autoSensedOpenShift 4 -}}
    {{/* Might indicate a configuration problem. */}}
    {{- include "srox.warn" (list $ "Detected user setting env.openshift=true, but auto-sensing of OpenShift version failed.") -}}
  {{- end -}}
  {{- $_ := set $env "openshift" 4 -}}
{{- else if and (kindIs "bool" $normalizedUserSpecifiedOpenShift) (not $normalizedUserSpecifiedOpenShift) -}}
  {{/* User set env.openshift=false. */}}
  {{- if gt $autoSensedOpenShift 0 -}}
    {{/* Might indicate a configuration problem. */}}
    {{- $fmt :=
          "User setting env.openshift=false is in contrast to the result of the auto-sensing, which yielded an OpenShift target cluster -- will proceed with user setting" -}}
    {{- include "srox.warn" (list $ $fmt) -}}
  {{- end -}}
  {{- $_ := set $env "openshift" 0 -}}
{{- else -}}
  {{/* User set env.openshift=4. */}}
  {{- if not (eq $normalizedUserSpecifiedOpenShift $autoSensedOpenShift) -}}
    {{/* Might indicate a configuration problem. */}}
    {{- $fmt := printf
          "User setting env.openshift=%d is in contrast to the result of the auto-sensing, which yielded a non-OpenShift cluster -- will proceed with user setting"
          $normalizedUserSpecifiedOpenShift -}}
    {{- include "srox.warn" (list $ $fmt) -}}
  {{- end -}}
  {{- $_ := set $env "openshift" $normalizedUserSpecifiedOpenShift -}}
{{- end -}}

{{- end -}}

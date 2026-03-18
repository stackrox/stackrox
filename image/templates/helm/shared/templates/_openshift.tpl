{{/*
    srox.autoSenseOpenshiftVersion $

    This function detects the OpenShift version automatically based on the cluster the Helm chart is installed onto.
    It writes the result to ._rox.env.openshift as an integer.
    Possible results are:
     - 4 (OpenShift 4)
     - 0 (Non-Openshift cluster)

    If "true" is passed for $._rox.env.openshift, this is unconditionally mapped to OpenShift version "4", because that is the only
    major version we currently support.
  */}}

{{ define "srox.autoSenseOpenshiftVersion" }}

{{ $ := index . 0 }}
{{ $env := $._rox.env }}

{{/* Infer OpenShift, if needed */}}
{{ if kindIs "invalid" $env.openshift }}
  {{/* This CRD API reliably indicates OpenShift 4. */}}
  {{ $_ := set $env "openshift" (has "config.openshift.io/v1" $._rox._apiServer.apiResources) }}
  {{- if $env.openshift -}}
    {{- include "srox.note" (list $ (printf "Based on API server properties, we have inferred that you are deploying into an OpenShift 4.x cluster.")) -}}
  {{- end -}}
{{ end }}
{{ if and $env.openshift (kindIs "bool" $env.openshift) }}
  {{/* We only support OpenShift 4. */}}
  {{ $_ := set $env "openshift" 4 }}
{{ end }}

{{ if not (kindIs "bool" $env.openshift) }}
  {{ $_ := set $env "openshift" (int $env.openshift) }}
{{ else if not $env.openshift }}
  {{ $_ := set $env "openshift" 0 }}
{{ end }}

{{- if and (ne $env.openshift 0) (ne $env.openshift 4) -}}
  {{- include "srox.fail" (printf "You have specified OpenShift version %d.x, but only version 4.x is currently supported." $env.openshift) -}}
{{- end -}}

{{ end }}

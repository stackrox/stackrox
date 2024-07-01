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

{{ define "srox.autoSenseOpenshiftVersion" }}

{{ $ := index . 0 }}
{{ $env := $._rox.env }}

{{/* Infer OpenShift, if needed */}}
{{ if kindIs "invalid" $env.openshift }}
  {{ $_ := set $env "openshift" (has "apps.openshift.io/v1" $._rox._apiServer.apiResources) }}
{{ end }}

{{/* Infer openshift version */}}
{{ if and $env.openshift (kindIs "bool" $env.openshift) }}
  {{/* Parse and add KubeVersion as semver from built-in resources. This is necessary to compare valid integer numbers. */}}
  {{ $kubeVersion := semver $.Capabilities.KubeVersion.Version }}

  {{/* Default to OpenShift 3 if no openshift resources are available, i.e. in helm template commands */}}
  {{ if not (has "apps.openshift.io/v1" $._rox._apiServer.apiResources) }}
    {{ $_ := set $._rox.env "openshift" 3 }}
  {{ else if gt $kubeVersion.Minor 11 }}
    {{ $_ := set $env "openshift" 4 }}
  {{ else }}
    {{ $_ := set $env "openshift" 3 }}
  {{ end }}
  {{ include "srox.note" (list $ (printf "Based on API server properties, we have inferred that you are deploying into an OpenShift %d.x cluster. Set the `env.openshift` property explicitly to 3 or 4 to override the auto-sensed value." $env.openshift)) }}
{{ end }}
{{ if not (kindIs "bool" $env.openshift) }}
  {{ $_ := set $env "openshift" (int $env.openshift) }}
{{ else if not $env.openshift }}
  {{ $_ := set $env "openshift" 0 }}
{{ end }}

{{ end }}

{{/*
  srox.setSecuredClusterCertRefresh

  Checks if Secured Cluster certificate refresh is enabled in the current context, and sets
  the securedClusterCertRefresh accordingly.

  It should be enabled for Helm and Operator installs, and only on Secured Clusters.
   */}}

{{ define "srox.setSecuredClusterCertRefresh" }}

{{ $ := index . 0 }}
{{ $env := $._rox.env }}

{{ if and (not $env.centralServices) (ne $env.installMethod "manifest") }}
  {{ $_ := set $env "securedClusterCertRefresh" true }}
{{ end }}

{{ end }}

{{- define "srox.securedClusterCertRefresh.certsPath" -}}
/tmp/certs
{{- end }}

{{- define "srox.securedClusterCertRefresh.newCertsPath" -}}
/tmp/certs-new
{{- end }}

{{- define "srox.securedClusterCertRefresh.legacyCertsPath" -}}
/tmp/certs-legacy
{{- end }}

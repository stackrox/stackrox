{{/*
  srox.setSecuredClusterCertRefresh

  Checks if Secured Cluster certificate refresh is enabled in the current context, and sets
  the securedClusterCertRefresh accordingly.

  It should be enabled for Helm and Operator installs, and only on Secured Clusters.
   */}}

{{ define "srox.setSecuredClusterCertRefresh" }}

{{ $ := index . 0 }}

{{ $_ := set $._rox "_securedClusterCertRefresh" false }}

{{ if and (not $._rox.env.centralServices) (ne $._rox.managedBy "MANAGER_TYPE_MANUAL") }}
  {{ $_ := set $._rox "_securedClusterCertRefresh" true }}
{{ end }}

{{ end }}

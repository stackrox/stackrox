{{/*
  srox.setInstallMethod $

  Sets $.env.installMethod to one of: "operator", "helm", "manifest".
*/}}

{{ define "srox.setInstallMethod" }}
{{ $ := index . 0 }}


{{ $_ := set $._rox.env "installMethod" "helm" }}

{{ end }}

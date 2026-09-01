{{/*
  srox.scannerDefaulting <Helm .> <Helm .Release> <Scanner configuration> <Stackrox Helm ConfigMap content>

  Encapsulates the Scanner defaulting logic.

  Can be removed later, together with StackRox Scanner.
*/}}

{{- define "srox.scannerDefaulting" -}}

{{- $ := index . 0 -}}
{{- $helmRelease := index . 1 -}}
{{- $scanner := index . 2 -}}
{{- $stackroxHelm := index . 3 -}}

{{/* The legacy StackRox Scanner (slim) is retired and is never installed, regardless of the
     scanner.disable setting. Warn if the user explicitly requested it, then coerce it off.
     Scanner V4 provides local (delegated) image scanning instead. */}}
{{- if and (not (kindIs "invalid" $scanner.disable)) (not $scanner.disable) -}}
  {{- include "srox.warn" (list $ "The StackRox Scanner is retired and will not be installed; Scanner V4 will be used for image scanning instead.") -}}
{{- end -}}
{{- $_ := set $scanner "disable" true -}}
{{- end -}}

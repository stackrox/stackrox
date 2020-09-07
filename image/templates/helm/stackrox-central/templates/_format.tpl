{{/*
  srox.formatStorageSize $value

  Formats $value as a storage size. $value can be an integer or a string.
  If no unit is specified (e.g., if $value is a string), a default unit of
  Gigabytes ("Gi" suffix) is assumed.
   */}}
{{- define "srox.formatStorageSize" -}}
{{- $val := toString . -}}
{{- if regexMatch "^[0-9]+$" $val -}}
  {{- $val = printf "%sGi" $val -}}
{{- end -}}
{{- default "0" $val -}}
{{- end -}}

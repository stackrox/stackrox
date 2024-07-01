{{- range $item := .items }}
{{- if eq $item.metadata.name "sensor-tls" }}
{{- $caPEM := index $item.data "ca.pem" }}
{{- if $caPEM }}
ca:
  cert: "{{ $caPEM | base64decode | js }}"
{{- end }}
{{- end }}
{{- end }}

{{- range $item := .items }}
{{- if eq $item.metadata.name "sensor-tls" }}
{{- $caPEM := index $item.data "ca.pem" }}
{{- if $caPEM }}
ca:
  cert: "{{ $caPEM | base64decode | js }}"
{{- end }}
{{- $sensorCert := index $item.data "sensor-cert.pem" }}
{{- $sensorKey := index $item.data "sensor-key.pem" }}
{{- if and $sensorCert $sensorKey }}
sensor:
  serviceTLS:
    cert: "{{ $sensorCert | base64decode | js }}"
    key: "{{ $sensorKey | base64decode | js }}"
{{- end }}
{{- else if eq $item.metadata.name "collector-tls" }}
{{- $collectorCert := index $item.data "collector-cert.pem" }}
{{- $collectorKey := index $item.data "collector-key.pem" }}
{{- if and $collectorCert $collectorKey }}
collector:
  serviceTLS:
    cert: "{{ $collectorCert | base64decode | js }}"
    key: "{{ $collectorKey | base64decode | js }}"
{{- end }}
{{- else if eq $item.metadata.name "admission-control-tls" }}
{{- $admCtrlCert := index $item.data "admission-control-cert.pem" }}
{{- $admCtrlKey := index $item.data "admission-control-key.pem" }}
{{- if and $admCtrlCert $admCtrlKey }}
admissionControl:
  serviceTLS:
    cert: "{{ $admCtrlCert | base64decode | js }}"
    key: "{{ $admCtrlKey | base64decode | js }}"
{{- end }}
{{- end }}
{{- end }}

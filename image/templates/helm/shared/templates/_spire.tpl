{{- define "srox.spireVolume" -}}
{{- $ := index . 0 -}}
{{- $env := $._rox.env -}}
{{- if $env.spire -}}
- name: spire-agent-socket
- hostPath:
    path: /run/spire/sockets
    type: Directory
{{- end -}}
{{- end -}}

{{- define "srox.spireVolumeMount" -}}
{{- $ := index . 0 -}}
{{- $env := $._rox.env -}}
{{- if $env.spire -}}
- name: spire-agent-socket
  mountPath: /run/spire/sockets
  readOnly: true
{{- end -}}
{{- end -}}

{{- define "srox.spirePodEnv" -}}
{{- $ := index . 0 -}}
{{- $env := $._rox.env -}}
{{- if $env.spire -}}
- name: SPIRE_AGENT_SOCKET
  value: /run/spire/sockets
{{- end -}}
{{- end -}}

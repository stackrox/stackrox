{{/*
  srox.injectedCABundleVolume

  Configures ConfigMap volume to use in a deployment.
   */}}
{{- define "srox.injectedCABundleVolume" -}}
{{- if eq ._rox.env.openshift 4 }}
- name: trusted-ca-volume
  configMap:
    name: injected-cabundle-{{ .Release.Name }}
    items:
      - key: ca-bundle.crt
        path: tls-ca-bundle.pem
    optional: true
{{ end }}
{{ end }}

{{/*
  srox.injectedCABundleVolumeMount

  Mounts the srox.injectedCABundle volume to a container.
   */}}
{{- define "srox.injectedCABundleVolumeMount" -}}
{{- if eq ._rox.env.openshift 4 }}
- name: trusted-ca-volume
  mountPath: /etc/pki/injected-ca-trust/
  readOnly: true
{{ end }}
{{ end }}

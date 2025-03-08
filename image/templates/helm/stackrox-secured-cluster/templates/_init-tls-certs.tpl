{{/*
    srox.tlsCertsInitContainer $

    This template produces the specification of the init container to be used
    for initializing the proper TLS certificates for the respective service.
   */}}
{{- define "srox.tlsCertsInitContainer" }}
{{- $ := index . 0 -}}
name: init-tls-certs
image: {{ quote $._rox.image.main.fullRef }}
command:
- /stackrox/bin/init-tls-certs
args:
- --legacy=/run/secrets/stackrox.io/certs-legacy/
- --new=/run/secrets/stackrox.io/certs-new/
- --destination=/run/secrets/stackrox.io/certs/
resources:
  requests:
    memory: "100Mi"
    cpu: "60m"
  limits:
    memory: "200Mi"
    cpu: "1000m"
securityContext:
  runAsNonRoot: true
  readOnlyRootFilesystem: true
volumeMounts:
- name: certs-legacy
  mountPath: /run/secrets/stackrox.io/certs-legacy/
  readOnly: true
- name: certs-new
  mountPath: /run/secrets/stackrox.io/certs-new/
  readOnly: true
- name: certs
  mountPath: /run/secrets/stackrox.io/certs/
{{- end }}

{{/*
    srox.tlsCertsInitContainerVolumes $serviceSlugName

    This template describes volumes needed by the init container for
    initializing TLS certificates. The service name is to be provided in its "slug form", e.g.
    "admission-control", "collector", etc.
   */}}
{{- define "srox.tlsCertsInitContainerVolumes" }}
{{- $serviceSlugName := index . 0 }}
- name: certs
  emptyDir: {}
- name: certs-legacy
  secret:
    secretName: {{ printf "%s-tls" $serviceSlugName | quote }}
    optional: true
    items:
    - key: {{ printf "%s-cert.pem" $serviceSlugName | quote }}
      path: cert.pem
    - key: {{ printf "%s-key.pem" $serviceSlugName | quote }}
      path: key.pem
    - key: ca.pem
      path: ca.pem
- name: certs-new
  secret:
    secretName: {{ printf "tls-cert-%s" $serviceSlugName | quote }}
    optional: true
{{- end }}

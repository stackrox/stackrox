{{/*
    srox.certWatcherSidecar $args

    This template produces the specification of a sidecar container that watches
    for TLS certificate changes and triggers a PostgreSQL reload via pg_ctl.

    Arguments (passed as a list):
      0: the full image reference for the postgres container
      1: the name of the volume containing the TLS certificates
   */}}
{{- define "srox.certWatcherSidecar" }}
{{- $image := index . 0 -}}
{{- $certVolumeName := index . 1 -}}
- name: cert-watcher
  image: {{ $image | quote }}
  command:
    - cert-watcher.sh
  env:
  - name: PGDATA
    value: "/var/lib/postgresql/data/pgdata"
  volumeMounts:
  - name: {{ $certVolumeName }}
    mountPath: /run/secrets/stackrox.io/certs
    readOnly: true
  - name: disk
    mountPath: /var/lib/postgresql/data
  resources:
    requests:
      cpu: 10m
      memory: 16Mi
    limits:
      cpu: 50m
      memory: 32Mi
  securityContext:
    allowPrivilegeEscalation: false
    runAsUser: 70
    runAsGroup: 70
{{- end }}

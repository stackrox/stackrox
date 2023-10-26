{{- /*
  This is the configuration file template for the Scanner v4 Indexer.
  Except for in extremely rare circumstances, you DO NOT need to modify this file.
  All config options that are possibly dynamic are templated out and can be modified
  via `--set`/values-files specified via `-f`.
     */ -}}

# Configuration file for Scanner v4 Indexer.
indexer:
  enable: true
  database:
    conn_string: "host=scanner-v4-db.{{ .Release.Namespace }}.svc port=5432 user=postgres sslmode={{- if eq .Release.Namespace "stackrox" }}verify-full{{- else }}verify-ca{{- end }} statement_timeout=60000"
  get_layer_timeout: 1m
matcher:
  enable: false
  database:
    conn_string: "host=scanner-v4-db.{{ .Release.Namespace }}.svc port=5432 user=postgres sslmode={{- if eq .Release.Namespace "stackrox" }}verify-full{{- else }}verify-ca{{- end }} statement_timeout=60000"
log_level: debug

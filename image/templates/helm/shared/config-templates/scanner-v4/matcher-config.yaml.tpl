{{- /*
  This is the configuration file template for the Scanner v4 Matcher.
  Except for in extremely rare circumstances, you DO NOT need to modify this file.
  All config options that are possibly dynamic are templated out and can be modified
  via `--set`/values-files specified via `-f`.
     */ -}}

# Configuration file for Scanner v4 Matcher.

indexer:
  centralEndpoint: https://central.{{ .Release.Namespace }}.svc
  postgres:
    # PostgreSQL Connection string
    # https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING
    conn: source: host=scanner-v4-db.{{ .Release.Namespace }}.svc port=5432 user=postgres sslmode={{- if eq .Release.Namespace "stackrox" }}verify-full{{- else }}verify-ca{{- end }} statement_timeout=60000

  api:
    httpsPort: 8080
    grpcPort: 8443

  updater:
    # Frequency with which the scanner will poll for vulnerability updates.
    interval: 5m

  logLevel: {{ ._rox.scanner.logLevel }}

  exposeMonitoring: false

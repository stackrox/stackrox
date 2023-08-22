{{- /*
  This is the configuration file template for the Scanner v4 Indexer.
  Except for in extremely rare circumstances, you DO NOT need to modify this file.
  All config options that are possibly dynamic are templated out and can be modified
  via `--set`/values-files specified via `-f`.
     */ -}}

# Configuration file for Scanner v4 Indexer.

indexer:
  centralEndpoint: https://central.{{ .Release.Namespace }}.svc
  sensorEndpoint: https://sensor.{{ .Release.Namespace }}.svc
  database:
    # Database driver
    type: pgsql
    options:
      # PostgreSQL Connection string
      # https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING
      source: host=scanner-v4-db.{{ .Release.Namespace }}.svc port=5432 user=postgres sslmode={{- if eq .Release.Namespace "stackrox" }}verify-full{{- else }}verify-ca{{- end }} statement_timeout=60000

      # Number of elements kept in the cache
      # Values unlikely to change (e.g. namespaces) are cached in order to save prevent needless roundtrips to the database.
      cachesize: 16384

  api:
    httpsPort: 8080
    grpcPort: 8443

  updater:
    # Frequency with which the scanner will poll for vulnerability updates.
    interval: 5m

  logLevel: {{ ._rox.scanner.logLevel }}

  exposeMonitoring: false

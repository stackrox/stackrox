{{- /*
  This is the configuration file template for Scanner.
  Except for in extremely rare circumstances, you DO NOT need to modify this file.
  All config options that are possibly dynamic are templated out and can be modified
  via `--set`/values-files specified via `-f`.
     */ -}}

# Configuration file for scanner.

scanner:
  database:
    # Database driver
    type: pgsql
    options:
      # PostgreSQL Connection string
      # https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING
      source: host=scanner-db.{{ .Release.Namespace }} port=5432 user=postgres sslmode=require statement_timeout=60000

      # Number of elements kept in the cache
      # Values unlikely to change (e.g. namespaces) are cached in order to save prevent needless roundtrips to the database.
      cachesize: 16384

  api:
    httpsPort: 8080
    grpcPort: 8081

  updater:
    # Frequency with which the scanner will poll for vulnerability updates.
    interval: 5m
    {{ if ._rox.env.offlineMode -}}
    fetchFromCentral: true
    {{- end }}

  logLevel: {{ ._rox.scanner.logLevel }}

  # The max size of files in images that are extracted. The scanner intentionally avoids extracting any files
  # larger than this to prevent DoS attacks. Leave commmented to use a reasonable default.
  # maxExtractableFileSizeMB: 200

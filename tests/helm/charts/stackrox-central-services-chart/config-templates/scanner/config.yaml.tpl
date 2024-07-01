{{- /*
  This is the configuration file template for Scanner.
  Except for in extremely rare circumstances, you DO NOT need to modify this file.
  All config options that are possibly dynamic are templated out and can be modified
  via `--set`/values-files specified via `-f`.
     */ -}}

# Configuration file for scanner.

scanner:
  centralEndpoint: https://central.{{ .Release.Namespace }}.svc
  sensorEndpoint: https://sensor.{{ .Release.Namespace }}.svc
  database:
    # Database driver
    type: pgsql
    options:
      # PostgreSQL Connection string
      # https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING
      source: host=scanner-db.{{ .Release.Namespace }}.svc port=5432 user=postgres sslmode={{- if eq .Release.Namespace "stackrox" }}verify-full{{- else }}verify-ca{{- end }} statement_timeout=60000

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

  # The scanner intentionally avoids extracting or analyzing any files
  # larger than the following default sizes to prevent DoS attacks.
  # Leave these commented to use a reasonable default.

  # The max size of files in images that are extracted.
  # Increasing this number increases memory pressure.
  # maxExtractableFileSizeMB: 200
  # The max size of ELF executable files that are analyzed.
  # Increasing this number may increase disk pressure.
  # maxELFExecutableFileSizeMB: 800
  # The max size of image file reader buffer. Image file data beyond this limit are overflowed to temporary files on disk.
  # maxImageFileReaderBufferSizeMB: 100

  exposeMonitoring: false

{{- /*
  This is the configuration file template for the Scanner v4 Matcher.
  Except for in extremely rare circumstances, you DO NOT need to modify this file.
  All config options that are possibly dynamic are templated out and can be modified
  via `--set`/values-files specified via `-f`.
     */ -}}

# Configuration file for Scanner v4 Matcher.
indexer:
  enable: false
matcher:
  enable: true
  database:
    conn_string: >
      host=scanner-v4-db.{{ .Release.Namespace }}.svc
      port=5432
      sslrootcert=/run/secrets/stackrox.io/certs/ca.pem
      user=postgres
      sslmode={{- if eq .Release.Namespace "stackrox" -}} verify-full {{- else -}} verify-ca {{- end }}
      {{ if not (kindIs "invalid" ._rox.scannerV4.db.source.statementTimeoutMs) -}} statement_timeout={{._rox.scannerV4.db.source.statementTimeoutMs}} {{- end }}
      {{ if not (kindIs "invalid" ._rox.scannerV4.db.source.minConns) -}} pool_min_conns={{._rox.scannerV4.db.source.minConns}} {{- end }}
      {{ if not (kindIs "invalid" ._rox.scannerV4.db.source.maxConns) -}} pool_max_conns={{._rox.scannerV4.db.source.maxConns}} {{- end }}
    password_file: /run/secrets/stackrox.io/secrets/password
  get_layer_timeout: 1m
log_level: debug

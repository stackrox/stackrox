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
      sslmode=verify-full
      {{ if not (kindIs "invalid" ._rox.scannerV4.db.source.statementTimeoutMs) -}} statement_timeout={{._rox.scannerV4.db.source.statementTimeoutMs}} {{- end }}
      {{ if not (kindIs "invalid" ._rox.scannerV4.db.source.minConns) -}} pool_min_conns={{._rox.scannerV4.db.source.minConns}} {{- end }}
      {{ if not (kindIs "invalid" ._rox.scannerV4.db.source.maxConns) -}} pool_max_conns={{._rox.scannerV4.db.source.maxConns}} {{- end }}
      client_encoding=UTF8
    password_file: /run/secrets/stackrox.io/secrets/password
  indexer_addr: scanner-v4-indexer.{{ .Release.Namespace }}.svc.cluster.local:8443
log_level: info

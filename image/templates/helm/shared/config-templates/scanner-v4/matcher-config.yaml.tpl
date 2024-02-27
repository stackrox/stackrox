{{- /*
  This is the configuration file template for the Scanner v4 Matcher.
  Except for in extremely rare circumstances, you DO NOT need to modify this file.
  All config options that are possibly dynamic are templated out and can be modified
  via `--set`/values-files specified via `-f`.
     */ -}}

# Configuration file for Scanner v4 Matcher.
stackrox_services: true
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
{{- if not (kindIs "invalid" ._rox.scannerV4.db.source.statementTimeoutMs) }}
      statement_timeout={{._rox.scannerV4.db.source.statementTimeoutMs}}
{{- end }}
{{- if not (kindIs "invalid" ._rox.scannerV4.db.source.minConns) }}
      pool_min_conns={{._rox.scannerV4.db.source.minConns}}
{{- end }}
{{- if not (kindIs "invalid" ._rox.scannerV4.db.source.maxConns) }}
      pool_max_conns={{._rox.scannerV4.db.source.maxConns}}
{{- end }}
      client_encoding=UTF8
    password_file: /run/secrets/stackrox.io/secrets/password
  vulnerabilities_url: https://central.{{ .Release.Namespace }}.svc/api/extensions/scannerdefinitions?version=ROX_VERSION
  indexer_addr: scanner-v4-indexer.{{ .Release.Namespace }}.svc:8443
log_level: "{{ ._rox.scannerV4.matcher.logLevel }}"
grpc_listen_addr: 0.0.0.0:8443
http_listen_addr: 0.0.0.0:9443
proxy:
  config_dir: /run/secrets/stackrox.io/proxy-config
  config_file: config.yaml

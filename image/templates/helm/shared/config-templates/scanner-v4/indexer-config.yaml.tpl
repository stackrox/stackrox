{{- /*
  This is the configuration file template for the Scanner v4 Indexer.
  Except for in extremely rare circumstances, you DO NOT need to modify this file.
  All config options that are possibly dynamic are templated out and can be modified
  via `--set`/values-files specified via `-f`.
     */ -}}

# Configuration file for Scanner v4 Indexer.
stackrox_services: true
indexer:
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
  get_layer_timeout: 1m
  {{- if ._rox.env.centralServices }}
  repository_to_cpe_url: https://central.{{ .Release.Namespace }}.svc/api/extensions/scannerdefinitions?file=repo2cpe
  name_to_repos_url: https://central.{{ .Release.Namespace }}.svc/api/extensions/scannerdefinitions?file=name2repos
  {{- else }}
  repository_to_cpe_url: https://sensor.{{ .Release.Namespace }}.svc/scanner/definitions?file=repo2cpe
  name_to_repos_url: https://sensor.{{ .Release.Namespace }}.svc/scanner/definitions?file=name2repos
  {{- end }}
  repository_to_cpe_file: /run/mappings/repository-to-cpe.json
  name_to_repos_file: /run/mappings/container-name-repos-map.json
matcher:
  enable: false
log_level: "{{ ._rox.scannerV4.indexer.logLevel }}"
grpc_listen_addr: 0.0.0.0:8443
http_listen_addr: 0.0.0.0:9443
proxy:
  config_dir: /run/secrets/stackrox.io/proxy-config
  config_file: config.yaml

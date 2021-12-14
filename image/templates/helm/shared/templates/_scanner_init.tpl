{{/*
  srox.scannerInit . $scannerConfig
   */}}

{{ define "srox.scannerInit" }}

{{ $ := index . 0 }}
{{ $scannerCfg := index . 1 }}

{{ include "srox.configureImage" (list $ $scannerCfg.image $._rox.image $._rox._state) }}
{{ include "srox.configureImage" (list $ $scannerCfg.dbImage $._rox.image $._rox._state) }}

{{ if or (eq $scannerCfg.mode "") (eq $scannerCfg.mode "full") }}
    {{ $scannerCertSpec := dict "CN" "SCANNER_SERVICE: Scanner" "dnsBase" "scanner" }}
    {{ include "srox.configureCrypto" (list $ "scanner.serviceTLS" $scannerCertSpec) }}

    {{ $scannerDBCertSpec := dict "CN" "SCANNER_DB_SERVICE: Scanner DB" "dnsBase" "scanner-db" }}
    {{ include "srox.configureCrypto" (list $ "scanner.dbServiceTLS" $scannerDBCertSpec) }}
{{ end }}

{{ include "srox.configurePassword" (list $ "scanner.dbPassword") }}

{{ end }}

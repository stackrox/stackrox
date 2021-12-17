{{/*
  srox.scannerInit . $scannerConfig

  Initializes the scanner configuration. The scanner chart has two modes "full" and
  "slim".
  The "full" mode is used for stand-alone deployments, mostly along with StackRox's Central service. In this
  mode the image contains vulnerability data and can create it's own certificates.

  The "slim" mode is used to deploy Scanner with a smaller image and does not generate TLS certificates.

  $scannerConfig contains all values which are configured by the user. The structure can be viewed in the according
  config-shape.
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

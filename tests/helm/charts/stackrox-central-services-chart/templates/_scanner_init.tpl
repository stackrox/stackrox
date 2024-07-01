{{/*
  srox.scannerInit . $scannerConfig

  Initializes the scanner configuration. The scanner chart has two modes "full" and
  "slim".
  The "full" mode is used for stand-alone deployments, mostly along with StackRox's Central service. In this
  mode, the image contains vulnerability data and the Helm chart can create its own certificates.

  The "slim" mode is used to deploy Scanner with a smaller image and does not generate TLS certificates,
  typically deployed within a Secured Cluster to scan images stored in a registry only accessible to the current cluster.
  The scanner chart defaults to "full" mode if no mode was provided.

  $scannerConfig contains all values which are configured by the user. The structure can be viewed in the according
  config-shape. See internal/scanner-config-shape.yaml.
   */}}

{{ define "srox.scannerInit" }}

{{ $ := index . 0 }}
{{ $scannerCfg := index . 1 }}

{{ if or (eq $scannerCfg.mode "") (eq $scannerCfg.mode "full") }}
    {{ include "srox.configureImage" (list $ $scannerCfg.image) }}
    {{ include "srox.configureImage" (list $ $scannerCfg.dbImage) }}

    {{ $scannerCertSpec := dict "CN" "SCANNER_SERVICE: Scanner" "dnsBase" "scanner" }}
    {{ include "srox.configureCrypto" (list $ "scanner.serviceTLS" $scannerCertSpec) }}

    {{ $scannerDBCertSpec := dict "CN" "SCANNER_DB_SERVICE: Scanner DB" "dnsBase" "scanner-db" }}
    {{ include "srox.configureCrypto" (list $ "scanner.dbServiceTLS" $scannerDBCertSpec) }}
{{ else if eq $scannerCfg.mode "slim" }}
    {{ include "srox.configureImage" (list $ $scannerCfg.slimImage) }}
    {{ include "srox.configureImage" (list $ $scannerCfg.slimDBImage) }}
{{ else }}
    {{ include "srox.fail" (printf "Unknown scanner mode %s" $scannerCfg.mode) }}
{{ end }}

{{ include "srox.configurePassword" (list $ "scanner.dbPassword") }}

{{ end }}

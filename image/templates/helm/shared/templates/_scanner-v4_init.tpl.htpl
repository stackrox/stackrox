{{/*
  srox.scannerV4Init . $scannerV4Config

  Initializes the Scanner v4 configuration. The scanner chart has two modes: Indexer and Matcher.
  In Indexer mode, the Scanner pulls images and analyzes them to determine the base OS and installed packages
  (i.e. it indexes the images).
  In Matcher mode, the Scanner matches the found packages to known vulnerabilities to produce a Vulnerability Report.
  Both modes require access to a PostgreSQL database.

  StackRox's Central service has two Scanner deployments: a Scanner running in Indexer mode and another
  running in Matcher mode. In this context, the Helm chart can create its own certificates.

  StackRox's Secured Cluster services may deploy the Scanner in Indexer mode, only.
  This would be done to access registries inaccessible to the Central cluster.
  In this context, the Helm chart does not generate its own certificates.

  $scannerV4Config contains all values which are configured by the user. The structures can be viewed in the respective
  config-shape. See internal/scanner-v4-config-shape.yaml.
   */}}

{{ define "srox.scannerV4Init" }}

{{ $ := index . 0 }}
{{ $scannerV4Cfg := index . 1 }}
{{ $_ := false }}

{{/* Sanity check. */}}
{{- if not (or (eq $.Chart.Name "stackrox-central-services") (eq $.Chart.Name "stackrox-secured-cluster-services")) -}}
  {{- include "srox.fail" (printf "Unexpected Helm chart name %q." $.Chart.Name) -}}
{{- end -}}

{{ $componentsCentralChart := dict "indexer" true "matcher" true }}
{{ $componentsSecuredClusterChart := dict "indexer" true "matcher" false }}

{{/* These will be propagated up. */}}
{{ $components := dict "indexer" false "matcher" false }}
{{ $dbEnabled  := false }}

{{- if not $scannerV4Cfg.disable }}
  {{/* Scanner V4 is switched on. */}}

  {{/* Scanner V4 component configuration depends on the chart. */}}
  {{- if eq $.Chart.Name "stackrox-central-services" -}}
    {{- $components = $componentsCentralChart -}}
  {{- else -}}
    {{- $components = $componentsSecuredClusterChart -}}
    {{/* scannerV4.indexer.disable can be used to disable the deployment of indexer.
         This is required for the operator use-case. */}}
    {{- if $scannerV4Cfg.indexer.disable -}}
      {{- $_ = set $components "indexer" false -}}
    {{- end -}}
  {{- end -}}

  {{/* Configure images, certificates and passwords as required. */}}
  {{ if or (get $components "indexer") (get $components "matcher") }}
    {{ include "srox.configureImage" (list $ $scannerV4Cfg.image) }}
  {{ end }}

  {{ if get $components "indexer" }}
    {{ $_ := set $scannerV4Cfg.indexer "image" $scannerV4Cfg.image }}
    {{- if eq $.Chart.Name "stackrox-central-services" -}}
      {{/* Only generate certificate when installing central-services.
           For secured-cluster-services we don't configure certificates here,
           instead they will be distributed at runtime by Sensor and Central. */}}
      {{- if kindIs "invalid" $._rox.scannerV4.indexer.serviceTLS.generate }}
        {{/* We need special handling here, because 'generate' will default to 'false' on upgrades. */}}
        {{/* And in case scanner V4 was not deployed earlier, we need to make sure that these resources are correctly initialized. */}}
        {{- if $.Release.IsUpgrade -}}
          {{- $lookupOut := dict -}}
          {{- $_ := include "srox.safeLookup" (list $ $lookupOut "v1" "Secret" $.Release.Namespace "scanner-v4-indexer-tls") -}}
          {{- if not $lookupOut.result -}}
            {{/* If generate=null and the resource does not exist yet, attempt to create it. */}}
            {{/* If lookup is not possible (e.g. in the operator), then 'generate' needs to be set correctly. */}}
            {{- $_ := set $._rox.scannerV4.indexer.serviceTLS "generate" true -}}
          {{- end -}}
        {{- end -}}
      {{- end }}
      {{ $cryptoSpec := dict "CN" "SCANNER_V4_INDEXER_SERVICE: Scanner V4 Indexer" "dnsBase" "scanner-v4-indexer" }}
      {{ include "srox.configureCrypto" (list $ "scannerV4.indexer.serviceTLS" $cryptoSpec) }}
    {{- end -}}
  {{ end }}

  {{ if get $components "matcher" }}
    {{ $_ := set $scannerV4Cfg.matcher "image" $scannerV4Cfg.image }}
    {{- if kindIs "invalid" $._rox.scannerV4.matcher.serviceTLS.generate }}
      {{/* We need special handling here, because 'generate' will default to 'false' on upgrades. */}}
      {{/* And in case scanner V4 was not deployed earlier, we need to make sure that these resources are correctly initialized. */}}
      {{- if $.Release.IsUpgrade -}}
        {{- $lookupOut := dict -}}
        {{- $_ := include "srox.safeLookup" (list $ $lookupOut "v1" "Secret" $.Release.Namespace "scanner-v4-matcher-tls") -}}
        {{- if not $lookupOut.result -}}
          {{/* If generate=null and the resource does not exist yet, attempt to create it. */}}
          {{/* If lookup is not possible (e.g. in the operator), then 'generate' needs to be set correctly. */}}
          {{- $_ := set $._rox.scannerV4.matcher.serviceTLS "generate" true -}}
        {{- end -}}
      {{- end }}
    {{- end }}
    {{ $cryptoSpec := dict "CN" "SCANNER_V4_MATCHER_SERVICE: Scanner V4 Matcher" "dnsBase" "scanner-v4-matcher" }}
    {{ include "srox.configureCrypto" (list $ "scannerV4.matcher.serviceTLS" $cryptoSpec) }}
  {{ end }}

  {{ if or (get $components "indexer") (get $components "matcher") }}
    {{- if kindIs "invalid" $._rox.scannerV4.db.serviceTLS.generate }}
      {{/* We need special handling here, because 'generate' will default to 'false' on upgrades. */}}
      {{/* And in case scanner V4 was not deployed earlier, we need to make sure that these resources are correctly initialized. */}}
      {{- if $.Release.IsUpgrade -}}
        {{- $lookupOut := dict -}}
        {{- $_ := include "srox.safeLookup" (list $ $lookupOut "v1" "Secret" $.Release.Namespace "scanner-v4-db-tls") -}}
        {{- if not $lookupOut.result -}}
          {{/* If generate=null and the resource does not exist yet, attempt to create it. */}}
          {{/* If lookup is not possible (e.g. in the operator), then 'generate' needs to be set correctly. */}}
          {{- $_ := set $._rox.scannerV4.db.serviceTLS "generate" true -}}
        {{- end -}}
      {{- end -}}
    {{- end }}
    {{- if kindIs "invalid" $._rox.scannerV4.db.password.generate }}
      {{/* We need special handling here, because 'generate' will default to 'false' on upgrades. */}}
      {{/* And in case scanner V4 was not deployed earlier, we need to make sure that these resources are correctly initialized. */}}
      {{- $lookupOut := dict -}}
      {{- $_ := include "srox.safeLookup" (list $ $lookupOut "v1" "Secret" $.Release.Namespace "scanner-v4-db-password") -}}
      {{- if not $lookupOut.result -}}
        {{/* If generate=null and the resource does not exist yet, attempt to create it. */}}
        {{/* If lookup is not possible (e.g. in the operator), then 'generate' needs to be set correctly. */}}
        {{- $_ := set $._rox.scannerV4.db.password "generate" true -}}
      {{- end -}}
    {{- end }}

    {{ include "srox.configureImage" (list $ $scannerV4Cfg.db.image) }}
    {{ include "srox.configurePassword" (list $ "scannerV4.db.password") }}

    {{- if eq $.Chart.Name "stackrox-central-services" -}}
      {{/* Only generate certificate when installing central-services. */}}
      {{ $cryptoSpec := dict "CN" "SCANNER_V4_DB_SERVICE: Scanner V4 DB" "dnsBase" "scanner-v4-db" }}
      {{ include "srox.configureCrypto" (list $ "scannerV4.db.serviceTLS" $cryptoSpec) }}
    {{- end -}}
  {{ end }}

{{- if eq $.Chart.Name "stackrox-secured-cluster-services" -}}
  {{/* Special handling for the secured-cluster-services chart in case it gets deployed
       to the same namespace as central-services. */}}
  {{ $centralDeployment := dict }}
  {{ include "srox.safeLookup" (list $ $centralDeployment "apps/v1" "Deployment" $.Release.Namespace "central") }}
  {{ if $centralDeployment.result }}
    {{ include "srox.note" (list $ "Detected central running in the same namespace. Not deploying scanner-v4-indexer from this chart and configuring sensor to use existing scanner-v4-indexer instance, if any.") }}
    {{ $_ := set $components "indexer" false }}
  {{ end }}
{{- end }}

{{- end -}} {{/* if not $scannerV4Cfg.disable */}}

{{/* Propagate information about which Scanner V4 components to deploy. */}}
{{- $_ := set $._rox "_scannerV4Enabled" (not $scannerV4Cfg.disable) -}}
{{- $_ := set $._rox.scannerV4 "_indexerEnabled" (get $components "indexer") -}}
{{- $_ := set $._rox.scannerV4 "_matcherEnabled" (get $components "matcher") -}}
{{- if or (get $components "indexer") (get $components "matcher") -}}
  {{- $_ := set $._rox.scannerV4 "_dbEnabled" true -}}
{{- end -}}

{{/* Provide some human-readable feedback regarding the Scanner V4 configuration to the user installing the Helm chart. */}}
{{- if and (get $components "indexer") (not (get $components "matcher")) -}}
  {{- $_ = set $._rox.scannerV4 "_installMode" "indexer" -}}
{{- else if and (not (get $components "indexer")) (get $components "matcher") -}}
  {{/* Just here for completeness, not allowed currently. */}}
  {{- $_ = set $._rox.scannerV4 "_installMode" "matcher" -}}
{{- else if and (get $components "indexer") (get $components "matcher") -}}
  {{- $_ = set $._rox.scannerV4 "_installMode" "indexer and matcher" -}}
{{- else -}}
  {{- $_ = set $._rox.scannerV4 "_installMode" "" -}}
{{- end -}}

{{- if not $scannerV4Cfg.disable }}
  {{- if eq $._rox.scannerV4._installMode "" }}
    {{ include "srox.note" (list $ (printf "Scanner V4 is enabled and Scanner V4 components are already deployed.")) }}
  {{- else }}
    {{ include "srox.note" (list $ (printf "Scanner V4 is enabled and the following Scanner V4 components will be deployed: %s" $._rox.scannerV4._installMode)) }}
  {{- end }}
{{- end }}

{{- end -}}

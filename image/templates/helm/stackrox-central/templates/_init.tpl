{{/*
    srox.init $

    Initialization template for the internal data structures.
    This template is designed to be included in every template file, but will only be executed
    once by leveraging state sharing between templates.
   */}}
{{ define "srox.init" }}

{{ $ := . }}

{{/*
    On first(!) instantiation, set up the $._rox structure, containing everything required by
    the resource template files.
   */}}
{{ if not $._rox }}

{{/*
    Initial Setup
   */}}

{{/*
    $rox / ._rox is the dictionary in which _all_ data that is modified by the init logic
    is stored.
    We ensure that it has the required shape, and then right after merging the user-specified
    $.Values, we apply some bootstrap defaults.
   */}}
{{ $rox := deepCopy $.Values }}
{{ $_ := include "srox.mergeInto" (list $rox ($.Files.Get "internal/config-shape.yaml" | fromYaml) ($.Files.Get "internal/bootstrap-defaults.yaml" | fromYaml)) }}
{{ $_ = set $ "_rox" $rox }}

{{/* Global state (accessed from sub-templates) */}}
{{ $generatedName := printf "stackrox-generated-%s" (randAlphaNum 6 | lower) }}
{{ $state := dict "customCertGen" false "generated" dict "generatedName" $generatedName "notes" list "warnings" list "referencedImages" dict }}
{{ $_ = set $._rox "_state" $state }}

{{/*
    General validation.
   */}}
{{ if ne $.Release.Namespace "stackrox" }}
  {{ if $._rox.allowNonstandardNamespace }}
    {{ include "srox.warn" (list $ "You have chosen to deploy to a namespace other than 'stackrox'. This might work, but is unsupported. Use with caution.") }}
  {{ else }}
    {{ include "srox.fail" (printf "You have chosen to deploy to namespace '%s', not 'stackrox'. If this was accidental, please re-run helm with the '-n stackrox' option. Otherwise, if you need to deploy into this namespace, set the 'allowNonstandardNamespace' configuration value to true." $.Release.Namespace) }}
  {{ end }}
{{ end }}

{{ if ne $.Release.Name $.Chart.Name }}
  {{ if $._rox.allowNonstandardReleaseName }}
    {{ include "srox.warn" (list $ (printf "You have chosen a release name of '%s', not '%s'. Accompanying scripts and commands in documentation might require adjustments." $.Release.Name $.Chart.Name)) }}
  {{ else }}
    {{ include "srox.fail" (printf "You have chosen a release name of '%s', not '%s'. We strongly recommend using the standard release name. If you must use a different name, set the 'allowNonstandardReleaseName' configuration option to true." $.Release.Name $.Chart.Name) }}
  {{ end }}
{{ end }}

{{/*
    API Server setup. The problem with `.Capabilities.APIVersions` is that Helm does not
    allow setting overrides for those when using `helm template` or `--dry-run`. Thus,
    if we rely on `.Capabilities.APIVersions` directly, we lose flexibility for our chart
    in these settings. Therefore, we use custom fields such that a user in principle has
    the option to inject via `--set`/`-f` everything we rely upon.
   */}}
{{ $apiResources := list }}
{{ if not (kindIs "invalid" $._rox.meta.apiServer.overrideAPIResources) }}
  {{ $apiResources = $._rox.meta.apiServer.overrideAPIResources }}
{{ else }}
  {{ range $apiResource := $.Capabilities.APIVersions }}
    {{ $apiResources = append $apiResources $apiResource }}
  {{ end }}
{{ end }}
{{ if $._rox.meta.apiServer.extraAPIResources }}
  {{ $apiResources = concat $apiResources $._rox.meta.apiServer.extraAPIResources }}
{{ end }}
{{ $apiServerVersion := coalesce $._rox.meta.apiServer.version $.Capabilities.KubeVersion.Version }}
{{ $apiServer := dict "apiResources" $apiResources "version" $apiServerVersion }}
{{ $_ = set $._rox "_apiServer" $apiServer }}


{{/*
    Environment setup - part 1
   */}}
{{ $env := $._rox.env }}

{{/* Infer OpenShift, if needed */}}
{{ if kindIs "invalid" $env.openshift }}
  {{ $_ := set $env "openshift" (has "apps.openshift.io/v1" $._rox._apiServer.apiResources) }}
  {{ if $env.openshift }}
    {{ include "srox.note" (list $ "Based on API server properties, we have inferred that you are deploying into an OpenShift cluster. Set the `env.openshift` property explicitly to false/true to override the auto-sensed value.") }}
  {{ end }}
{{ end }}

{{/* Infer Istio, if needed */}}
{{ if kindIs "invalid" $env.istio }}
  {{ $_ := set $env "istio" (has "networking.istio.io/v1alpha3" $._rox._apiServer.apiResources) }}
  {{ if $env.istio }}
    {{ include "srox.note" (list $ "Based on API server properties, we have inferred that you are deploying into an Istio-enabled cluster. Set the `env.istio` property explicitly to false/true to override the auto-sensed value.") }}
  {{ end }}
{{ end }}

{{/* Infer GKE, if needed */}}
{{ if kindIs "invalid" $env.platform }}
  {{ $platform := "default" }}
  {{ if contains "-gke." $._rox._apiServer.version }}
    {{ include "srox.note" (list $ "Based on API server properties, we have inferred that you are deploying into a GKE cluster. Set the `env.platform` property to a concrete value to override the auto-sensed value.") }}
    {{ $platform = "gke" }}
  {{ end }}
  {{ $_ := set $env "platform" $platform }}
{{ end }}

{{/* Apply defaults */}}
{{ $defaultsCfg := dict }}
{{ $platformCfgFile := dict }}
{{ include "srox.loadFile" (list $ $platformCfgFile (printf "internal/platforms/%s.yaml" $env.platform)) }}
{{ if not $platformCfgFile.found }}
  {{ include "srox.fail" (printf "Invalid platform %q. Please select a valid platform, or leave this field unset." $env.platform) }}
{{ end }}
{{ $_ = include "srox.mergeInto" (list $defaultsCfg (fromYaml $platformCfgFile.contents) ($.Files.Get "internal/defaults.yaml" | fromYaml)) }}
{{ $_ = set $rox "_defaults" $defaultsCfg }}
{{ $_ = include "srox.mergeInto" (list $rox $defaultsCfg.defaults) }}


{{/* Expand applicable config values */}}
{{ $expandables := $.Files.Get "internal/expandables.yaml" | fromYaml }}
{{ include "srox.expandAll" (list $ $rox $expandables) }}


{{/* Image pull secret setup. */}}
{{ $imagePullSecrets := $._rox.imagePullSecrets }}
{{ $imagePullSecretNames := default list $imagePullSecrets.useExisting }}
{{ if not (kindIs "slice" $imagePullSecretNames) }}
  {{ $imagePullSecretNames = regexSplit "\\s*,\\s*" (trim $imagePullSecretNames) -1 }}
{{ end }}
{{ if $imagePullSecrets.useFromDefaultServiceAccount }}
  {{ $defaultSA := dict }}
  {{ include "srox.safeLookup" (list $ $defaultSA "v1" "ServiceAccount" $.Release.Namespace "default") }}
  {{ if $defaultSA.result }}
    {{ $imagePullSecretNames = concat $imagePullSecretNames (default list $defaultSA.result.imagePullSecrets) }}
  {{ end }}
{{ end }}
{{ $imagePullCreds := dict }}
{{ if $imagePullSecrets._username }}
  {{ $imagePullCreds = dict "username" $imagePullSecrets._username "password" $imagePullSecrets._password }}
  {{ $imagePullSecretNames = append $imagePullSecretNames "stackrox" }}
{{ else if $imagePullSecrets._password }}
  {{ include "srox.fail" "Whenever an image pull password is specified, a username must be specified as well "}}
{{ end }}
{{ if and $.Release.IsInstall (not $imagePullSecretNames) (not $imagePullSecrets.allowNone) }}
  {{ include "srox.fail" "You have not specified any image pull secrets, and no existing image pull secrets were automatically inferred. If your registry does not need image pull credentials, explicitly set the 'imagePullSecrets.allowNone' option to 'true'" }}
{{ end }}

{{/*
    Always assume that there are `stackrox` and `stackrox-scanner` image pull secrets,
    even if they weren't specified.
    This is required for updates anyway, so referencing it on first install will minimize a later
    diff.
   */}}
{{ $imagePullSecretNames = concat $imagePullSecretNames (list "stackrox" "stackrox-scanner") | uniq | sortAlpha }}
{{ $_ = set $imagePullSecrets "_names" $imagePullSecretNames }}
{{ $_ = set $imagePullSecrets "_creds" $imagePullCreds }}


{{/* Global CA setup */}}
{{ $caCertSpec := dict "CN" "StackRox Certificate Authority" "ca" true }}
{{ include "srox.configureCrypto" (list $ "ca" $caCertSpec) }}


{{/* Proxy configuration.
     Note: The reason this is different is that unlike the endpoints config, the proxy configuration
     might contain sensitive data and thus might _not_ be stored in the always available canonical
     values file. However, this is probably rare. Therefore, for this particular instance we do decide
     to rely on lookup magic for initially populating the secret with a default proxy config.
     However, we won't take any chances, and therefore only create that secret if we can be reasonably
     confident that lookup actually works, by trying to lookup the default service account.
   */}}
{{ $proxyCfg := $env._proxyConfig }}
{{ $fileOut := dict }}
{{ include "srox.loadFile" (list $ $fileOut "config/proxy-config.yaml") }}
{{ if $fileOut.found }}
  {{ if not (kindIs "invalid" $proxyCfg) }}
    {{ include "srox.fail" "Both env.proxyConfig was specified, and a config/proxy-config.yaml was found. Please remove/rename the config file, or comment out the env.proxyConfig stanza." }}
  {{ end }}
  {{ $proxyCfg = $fileOut.contents }}
{{ end }}

{{/* On first install, create a default proxy config, but only if we can be sure none exists. */}}
{{ if and (kindIs "invalid" $proxyCfg) $.Release.IsInstall }}
  {{ $lookupOut := dict }}
  {{ include "srox.safeLookup" (list $ $lookupOut "v1" "Secret" $.Release.Namespace "proxy-config") }}
  {{ if and $lookupOut.reliable (not $lookupOut.result) }}
    {{ $fileOut := dict }}
    {{ include "srox.loadFile" (list $ $fileOut "config/proxy-config.yaml.default") }}
    {{ $proxyCfg = $fileOut.contents }}
  {{ end }}
{{ end }}
{{ $_ = set $env "_proxyConfig" $proxyCfg }}


{{/*
    Central setup.
   */}}

{{ $centralCfg := $rox.central }}

{{/* Image settings */}}
{{ if kindIs "invalid" $centralCfg.image.tag }}
  {{ $_ := set $centralCfg.image "tag" $.Chart.AppVersion }}
{{ end }}
{{ include "srox.configureImage" (list $ $centralCfg.image) }}

{{/* Admin password */}}
{{ include "srox.configurePassword" (list $ "central.adminPassword" "admin") }}

{{/* Service TLS Certificates */}}
{{ $centralCertSpec := dict "CN" "CENTRAL_SERVICE: Central" "dnsBase" "central" }}
{{ include "srox.configureCrypto" (list $ "central.serviceTLS" $centralCertSpec) }}

{{/* JWT Token Signer */}}
{{ $jwtSignerSpec := dict "keyOnly" "rsa" }}
{{ include "srox.configureCrypto" (list $ "central.jwtSigner" $jwtSignerSpec) }}

{{/* License Key */}}
{{/* Note: this is at the top-level in $.Values, but this is purely to achieve a less surprising
     user interface. It effectively is part of the Central configuration. */}}
{{ $licenseKey := $._rox._licenseKey }}
{{ if and (not $licenseKey) $.Release.IsInstall }}
  {{/* Even on install, check if there might be a pre-existing license key to minimize confusion. */}}
  {{ $licenseLookupOut := dict }}
  {{ include "srox.safeLookup" (list $ $licenseLookupOut "v1" "Secret" $.Release.Namespace "central-license") }}
  {{ if not $licenseLookupOut.result }}
    {{ include "srox.warn" (list $ "No StackRox license provided. Make sure a valid license exists in Kubernetes secret 'central-license'.") }}
  {{ end }}
{{ end }}

{{/* Setup Default TLS Certificate. */}}
{{ if $._rox.central.defaultTLS }}
  {{ $cert := $._rox.central.defaultTLS._cert }}
  {{ $key := $._rox.central.defaultTLS._key }}
  {{ if and $cert $key }}
    {{ $defaultTLSCert := dict "Cert" $cert "Key" $key }}
    {{ $_ := set $._rox.central "_defaultTLS" $defaultTLSCert }}
    {{ include "srox.note" (list $ "Configured default TLS certificate") }}
  {{ else if or $cert $key }}
    {{ include "srox.fail" "Must specify either none or both of central.defaultTLS.cert and central.defaultTLS.key" }}
  {{ end }}
{{ end }}

{{/*
    Setup configuration for persistence backend.
  */}}
{{ $volumeCfg := dict }}
{{ if $centralCfg.persistence.none }}
  {{ include "srox.warn" (list $ "You have selected no persistence backend. Every deletion of the StackRox Central pod will cause you to lose all your data. This is STRONGLY recommended against.") }}
  {{ $_ := set $volumeCfg "emptyDir" dict }}
{{ end }}
{{ if $centralCfg.persistence.hostPath }}
  {{ if not $centralCfg.nodeSelector }}
    {{ include "srox.warn" (list $ "You have selected host path persistence, but not specified a node selector. This is unlikely to work reliably.") }}
  {{ end }}
  {{ $_ := set $volumeCfg "hostPath" (dict "path" $centralCfg.persistence.hostPath) }}
{{ end }}
{{/* Configure PVC if either any of the settings in `central.persistence.persistentVolumeClaim` are non-zero,
     or no other persistence backend has been configured yet. */}}
{{ if or (and $centralCfg.persistence.persistentVolumeClaim (values $centralCfg.persistence.persistentVolumeClaim | compact)) (not $volumeCfg) }}
  {{ $pvcCfg := $centralCfg.persistence.persistentVolumeClaim }}
  {{ $_ := include "srox.mergeInto" (list $pvcCfg $._rox._defaults.pvcDefaults (dict "createClaim" $.Release.IsInstall)) }}
  {{ $_ = set $volumeCfg "persistentVolumeClaim" (dict "claimName" $pvcCfg.claimName) }}
  {{ if $pvcCfg.createClaim }}
    {{ $_ = set $centralCfg.persistence "_pvcCfg" $pvcCfg }}
  {{ end }}
{{ end }}

{{ $allPersistenceMethods := keys $volumeCfg | sortAlpha }}
{{ if ne (len $allPersistenceMethods) 1 }}
  {{ include "srox.fail" (printf "Invalid or no persistence configurations for central: [%s]" (join "," $allPersistenceMethods)) }}
{{ end }}
{{ $_ = set $centralCfg.persistence "_volumeCfg" $volumeCfg }}


{{/*
    Exposure configuration setup & sanity checks.
   */}}
{{ if $._rox.central.exposure.loadBalancer.enabled }}
  {{ include "srox.note" (list $ (printf "Exposing StackRox Central via LoadBalancer service.")) }}
{{ end }}
{{ if $._rox.central.exposure.nodePort.enabled }}
  {{ include "srox.note" (list $ (printf "Exposing StackRox Central via NodePort service.")) }}
{{ end }}
{{ if $._rox.central.exposure.route.enabled }}
  {{ if not $env.openshift }}
    {{ include "srox.fail" (printf "The exposure method 'Route' is only available on OpenShift clusters.") }}
  {{ end }}
  {{ include "srox.note" (list $ (printf "Exposing StackRox Central via OpenShift Route https://central.%s." $.Release.Namespace)) }}
{{ end }}

{{ if not (or $._rox.central.exposure.loadBalancer.enabled $._rox.central.exposure.nodePort.enabled $._rox.central.exposure.route.enabled) }}
  {{ include "srox.note" (list $ "Not exposing StackRox Central, it will only be reachable cluster-internally.") }}
  {{ include "srox.note" (list $ "To enable exposure via LoadBalancer service, use --set central.exposure.loadBalancer.enabled=true.") }}
  {{ include "srox.note" (list $ "To enable exposure via NodePort service, use --set central.exposure.nodePort.enabled=true.") }}
  {{ if $env.openshift }}
    {{ include "srox.note" (list $ "To enable exposure via an OpenShift Route, use --set central.exposure.route.enabled=true.") }}
  {{ end }}
  {{ include "srox.note" (list $ (printf "To acccess StackRox Central via a port-forward on your local port 18443, run: kubectl -n %s port-forward svc/central 18443:443." .Release.Namespace)) }}
{{ end }}

{{/*
    Scanner setup.
   */}}

{{ $scannerCfg := $._rox.scanner }}

{{ if and $scannerCfg.disable (or $.Release.IsInstall $.Release.IsUpgrade) }}
  {{/* We generally don't recommend customers run without scanner, so show a warning to the user */}}
  {{ $action := ternary "deploy StackRox Central Services without Scanner" "upgrade StackRox Central Services without Scanner (possibly removing an existing Scanner deployment)" $.Release.IsInstall }}
  {{ include "srox.warn" (list $ (printf "You have chosen to %s. Certain features dependent on image scanning might not work." $action)) }}
{{ else if not $scannerCfg.disable }}
  {{ include "srox.configureImage" (list $ $scannerCfg.image) }}
  {{ include "srox.configureImage" (list $ $scannerCfg.dbImage) }}

  {{ $scannerCertSpec := dict "CN" "SCANNER_SERVICE: Scanner" "dnsBase" "scanner" }}
  {{ include "srox.configureCrypto" (list $ "scanner.serviceTLS" $scannerCertSpec) }}

  {{ $scannerDBCertSpec := dict "CN" "SCANNER_DB_SERVICE: Scanner DB" "dnsBase" "scanner-db" }}
  {{ include "srox.configureCrypto" (list $ "scanner.dbServiceTLS" $scannerDBCertSpec) }}

  {{ include "srox.configurePassword" (list $ "scanner.dbPassword") }}
{{ end }}


{{/*
    Post-processing steps.
   */}}


{{/* Compact the post-processing config to prevent it from appearing non-empty if it doesn't
     contain any concrete (leaf) values. */}}
{{ include "srox.compactDict" (list $._rox._state.generated -1) }}

{{/* Setup Image Pull Secrets for Docker Registry.
     Note: This must happen afterwards, as we rely on "srox.configureImage" to collect the
     set of all referenced images first. */}}
{{ if $imagePullSecrets._username }}
  {{ $dockerAuths := dict }}
  {{ range $image := keys $._rox._state.referencedImages }}
    {{ $registry := splitList "/" $image | first }}
    {{ if eq $registry "docker.io" }}
      {{/* Special case docker.io */}}
      {{ $registry = "https://index.docker.io/v1/" }}
    {{ else }}
      {{ $registry = printf "https://%s" $registry }}
    {{ end }}
    {{ $_ := set $dockerAuths $registry dict }}
  {{ end }}
  {{ $authToken := printf "%s:%s" $imagePullSecrets._username $imagePullSecrets._password | b64enc }}
  {{ range $regSettings := values $dockerAuths }}
    {{ $_ := set $regSettings "auth" $authToken }}
  {{ end }}

  {{ $_ := set $imagePullSecrets "_dockerAuths" $dockerAuths }}
{{ end }}

{{/* Final warnings based on state. */}}
{{ if $._rox._state.customCertGen }}
  {{ include "srox.warn" (list $ "At least one certificate was generated by Helm. Helm limits the generation of custom certificates to RSA private keys, which have poorer computational performance. Consider using roxctl for certificate generation of certificates with ECDSA private keys for improved performance. (THIS IS NOT A SECURITY ISSUE)") }}
{{ end }}

{{ end }}
{{ end }}

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
    Currently we only expose the old-style configuration schema,
    even though chart-internally we already use the new configuration schema.
    Here we do the necessary transformation.
   */}}

{{/* BEGIN: LEGACY FORMAT -> NEW FORMAT TRANSFORMATION. */}}

{{ $_compatibilityMode := true }}
{{ $_values := dict }}
{{ $_ := include "srox.mergeInto" (list $_values (deepCopy .Values) ($.Files.Get "internal/compatibility-config-shape.yaml" | fromYaml)) }}

{{/* ImagePullSecrets config. */}}
{{ $_imagePullSecrets := dict "allowNone" true }}
{{ $_env := dict }}
{{ if not (kindIs "invalid" $_values.cluster.type) }}
  {{ $_ := set $_env "openshift" (eq $_values.cluster.type "OPENSHIFT_CLUSTER") }}
{{ end }}

{{/* CA config. */}}
{{ $_ca := dict }}
{{ if $_values.config.createSecrets }}
  {{ $_ := set $_ca "cert" "@?secrets/ca.pem" }}
{{ else }}
  {{ $_ := set $_ca "_cert" "" }}
{{ end }}

{{/* Main Image. */}}
{{ $_mainImage := dict }}
{{ $_ := set $_mainImage "registry" $_values.image.registry.main }}
{{ $_ := set $_mainImage "name" $_values.image.repository.main }}

{{/* Sensor config. */}}
{{ $_sensor := dict }}
{{ $_sensorServiceTLS := dict }}
{{ if $_values.config.createSecrets }}
  {{ $_ := set $_sensorServiceTLS "cert" "@?secrets/sensor-cert.pem" }}
  {{ $_ := set $_sensorServiceTLS "key" "@?secrets/sensor-key.pem" }}
{{ else }}
  {{ $_ := set $_sensorServiceTLS "_cert" "" }}
  {{ $_ := set $_sensorServiceTLS "_key" "" }}
{{ end }}
{{ $_ := set $_sensor "image" $_mainImage }}

{{ $_ := set $_sensor "serviceTLS" $_sensorServiceTLS }}
{{ if not (kindIs "invalid" $_values.endpoint.advertised) }}
  {{ $_ := set $_sensor "endpoint" $_values.endpoint.advertised }}
{{ end }}

{{/* AdmissionControl config. */}}
{{ $_admissionControl := dict }}
{{ $_admissionControlServiceTLS := dict }}
{{ if $_values.config.createSecrets }}
  {{ $_ := set $_admissionControlServiceTLS "cert" "@?secrets/admission-control-cert.pem" }}
  {{ $_ := set $_admissionControlServiceTLS "key" "@?secrets/admission-control-key.pem" }}
{{ else }}
  {{ $_ := set $_admissionControlServiceTLS "_cert" "" }}
  {{ $_ := set $_admissionControlServiceTLS "_key" "" }}
{{ end }}
{{ $_ := set $_admissionControl "enable" $_values.config.admissionControl.createService }}
{{ $_ := set $_admissionControl "listenOnUpdates" $_values.config.admissionControl.listenOnUpdates }}
{{ $_ := set $_admissionControl "enforceOnUpdates" $_values.config.admissionControl.enforceOnUpdates }}
{{ $_ := set $_admissionControl "dynamic" (dict "enforce" $_values.config.admissionControl.enableService "scanInline" $_values.config.admissionControl.scanInline "disableBypass" $_values.config.admissionControl.disableBypass "timeout" $_values.config.admissionControl.timeout) }}
{{ $_ := set $_admissionControl "image" $_mainImage }}
{{ $_ := set $_admissionControl "serviceTLS" $_admissionControlServiceTLS }}

{{/* Collector config. */}}
{{ $_collector := dict }}
{{ $_collectorImage := dict }}
{{ $_ := set $_collectorImage "registry" $_values.image.registry.collector }}
{{ $_ := set $_collectorImage "name" $_values.image.repository.collector }}
{{ $_collectorServiceTLS := dict }}
{{ if $_values.config.createSecrets }}
  {{ $_ := set $_collectorServiceTLS "cert" "@?secrets/collector-cert.pem" }}
  {{ $_ := set $_collectorServiceTLS "key" "@?secrets/collector-key.pem" }}
{{ else }}
  {{ $_ := set $_collectorServiceTLS "_cert" "" }}
  {{ $_ := set $_collectorServiceTLS "_key" "" }}
{{ end }}
{{ $_ := set $_collector "image" $_collectorImage }}
{{ $_ := set $_collector "collectionMethod" $_values.config.collectionMethod }}
{{ $_ := set $_collector "disableTaintTolerations" $_values.config.disableTaintTolerations }}
{{ $_ := set $_collector "slimMode" $_values.config.slimCollector }}
{{ $_ := set $_collector "serviceTLS" $_collectorServiceTLS }}

{{/* Additional CAs. */}}
{{ $_additionalCAs := dict }}
{{ range $path, $content := .Files.Glob "secrets/additional-cas/**" }}
  {{ $_ := set $_additionalCAs (base $path) (toString $content) }}
{{ end }}

{{/* General (new-style) customization plus support for old-style envVars style config. */}}
{{ $_customize := dict }}
{{ if kindIs "map" $_values.customize }}
  {{ $_customize = deepCopy $_values.customize }}
{{ end }}
{{ if kindIs "invalid" $_customize.envVars }}
  {{ $_ := set $_customize "envVars" dict }}
{{ end }}
{{ range $_values.envVars }}
  {{ $_ := set $_customize.envVars .name .value }}
{{ end }}

{{/* Transform into new-style config. */}}
{{ $newValues := dict }}
{{ $_ := set $newValues "clusterName" $_values.cluster.name }}
{{ $_ := set $newValues "centralEndpoint" $_values.endpoint.central }}
{{ $_ := set $newValues "imagePullSecrets" $_imagePullSecrets }}
{{ $_ := set $newValues "ca" $_ca }}
{{ $_ := set $newValues "additionalCAs" $_additionalCAs }}
{{ $_ := set $newValues "env" $_env }}
{{ $_ := set $newValues "customize" $_customize }}
{{ $_ := set $newValues "sensor" $_sensor }}
{{ $_ := set $newValues "admissionControl" $_admissionControl }}
{{ $_ := set $newValues "collector" $_collector }}
{{ if not (kindIs "invalid" $_values.config.createUpgraderServiceAccount) }}
  {{ $_ := set $newValues "createUpgraderServiceAccount" $_values.config.createUpgraderServiceAccount }}
{{ end }}
{{ $_ := set $ "Values" $newValues }}

{{/* END: LEGACY FORMAT -> NEW FORMAT TRANSFORMATION. */}}

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
{{ $state := dict "notes" list "warnings" list "referencedImages" dict }}
{{ $_ = set $._rox "_state" $state }}

{{/*
    General validation.
   */}}
{{ if not $_compatibilityMode }}

{{ if ne $.Release.Namespace "stackrox" }}
  {{ if $._rox.allowNonstandardNamespace }}
    {{ include "srox.note" (list $ (printf "You have chosen to deploy to namespace '%s'." $.Release.Namespace)) }}
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

{{/* Infer platform. */}}
{{ if kindIs "invalid" $env.platform }}
  {{ $platform := "default" }}
  {{/* Platform specific configuration can be injected here, currently we only use "default" for this chart. */}}
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
{{ $imagePullSecretNames = concat $imagePullSecretNames (list "stackrox") | uniq | sortAlpha }}
{{ $_ = set $imagePullSecrets "_names" $imagePullSecretNames }}
{{ $_ = set $imagePullSecrets "_creds" $imagePullCreds }}


{{/*
    Sensor setup.
   */}}

{{ $sensorCfg := $rox.sensor }}

{{/* Image settings */}}
{{ if kindIs "invalid" $sensorCfg.image.tag }}
  {{ $_ := set $sensorCfg.image "tag" $.Chart.AppVersion }}
{{ end }}
{{ include "srox.configureImage" (list $ $sensorCfg.image) }}

{{/*
    Post-processing steps.
   */}}


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

{{ end }}

{{ end }}

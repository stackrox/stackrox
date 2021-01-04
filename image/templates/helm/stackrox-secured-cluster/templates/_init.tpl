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
    Calculate the fingerprint of the input config.
   */}}
{{ $configFP := printf "%s-%d" (.Values | toJson | sha256sum) .Release.Revision }}

{{/*
    Initial Setup
   */}}

{{/*
    Currently we only expose the old-style configuration schema,
    even though chart-internally we already use the new configuration schema.
    Here we do the necessary transformation.
   */}}


{{ $_compatibilityMode := false }}

{{ if kindIs "invalid" .Values.initToken }}

{{/* BEGIN: LEGACY FORMAT -> NEW FORMAT TRANSFORMATION. */}}
{{ $_compatibilityMode = true }}
{{ $_values := dict }}
{{ $_ := include "srox.mergeInto" (list $_values (deepCopy .Values) ($.Files.Get "internal/compatibility-config-shape.yaml" | fromYaml)) }}

{{/* ImagePullSecrets config (for main and collector images). */}}
{{ $_mainImagePullSecrets := dict }}
{{ $_ := include "srox.mergeInto" (list $_mainImagePullSecrets (deepCopy $_values.mainImagePullSecrets) (deepCopy $_values.imagePullSecrets)) }}
{{ if and (kindIs "invalid" $_mainImagePullSecrets.allowNone) (kindIs "invalid" $_mainImagePullSecrets.useExisting) }}
  {{ $_ := set $_mainImagePullSecrets "allowNone" true }}
{{ end }}

{{ $_collectorImagePullSecrets := dict }}
{{ $_ := include "srox.mergeInto" (list $_collectorImagePullSecrets (deepCopy $_values.collectorImagePullSecrets) (deepCopy $_values.imagePullSecrets)) }}
{{ if and (kindIs "invalid" $_collectorImagePullSecrets.allowNone) (kindIs "invalid" $_collectorImagePullSecrets.useExisting) }}
  {{ $_ := set $_collectorImagePullSecrets "allowNone" true }}
{{ end }}

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
{{ $_ := set $_mainImage "pullPolicy" $_values.image.pullPolicy.main }}
{{ $_ := set $_mainImage "tag" $_values.image.tag.main }}

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
{{ $_ := set $_sensor "resources" $_values.config.sensorResources }}
{{ $_ := set $_sensor "serviceTLS" $_sensorServiceTLS }}
{{ if not (kindIs "invalid" $_values.endpoint.advertised) }}
  {{ $_ := set $_sensor "endpoint" $_values.endpoint.advertised }}
{{ end }}
{{ $_ := set $_sensor "exposeMonitoring" $_values.config.exposeMonitoring }}

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

{{ $_admissionControlDynamic := dict }}
{{ $_ := set $_admissionControlDynamic "enforce" $_values.config.admissionControl.enableService }}
{{ $_ := set $_admissionControlDynamic "scanInline" $_values.config.admissionControl.scanInline }}
{{ $_ := set $_admissionControlDynamic "disableBypass" $_values.config.admissionControl.disableBypass }}
{{ $_ := set $_admissionControlDynamic "timeout" $_values.config.admissionControl.timeout }}
{{ $_ := set $_admissionControlDynamic "enforceOnUpdates" $_values.config.admissionControl.enforceOnUpdates }}

{{ $_ := set $_admissionControl "enable" $_values.config.admissionControl.createService }}
{{ $_ := set $_admissionControl "listenOnUpdates" $_values.config.admissionControl.listenOnUpdates }}
{{ $_ := set $_admissionControl "enforceOnUpdates" $_values.config.admissionControl.enforceOnUpdates }}
{{ $_ := set $_admissionControl "image" $_mainImage }}
{{ $_ := set $_admissionControl "resources" $_values.config.admissionControlResources }}
{{ $_ := set $_admissionControl "serviceTLS" $_admissionControlServiceTLS }}
{{ $_ := set $_admissionControl "exposeMonitoring" $_values.config.exposeMonitoring }}
{{ $_ := set $_admissionControl "dynamic" $_admissionControlDynamic }}

{{/* Collector Image. */}}
{{ $_collectorImage := dict }}
{{ $_ := set $_collectorImage "registry" $_values.image.registry.collector }}
{{ $_ := set $_collectorImage "name" $_values.image.repository.collector }}
{{ if kindIs "invalid" $_values.image.pullPolicy.collector }}
  {{ if $_values.config.slimCollector }}
    {{ $_ := set $_collectorImage "pullPolicy" "IfNotPresent" }}
  {{ else }}
    {{ $_ := set $_collectorImage "pullPolicy" "Always" }}
  {{ end }}
{{ else }}
  {{ $_ := set $_collectorImage "pullPolicy" $_values.image.pullPolicy.collector }}
{{ end }}
{{ $_ := set $_collectorImage "tag" $_values.image.tag.collector }}

{{/* Collector config. */}}
{{ $_collector := dict }}
{{ $_collectorServiceTLS := dict }}
{{ if $_values.config.createSecrets }}
  {{ $_ := set $_collectorServiceTLS "cert" "@?secrets/collector-cert.pem" }}
  {{ $_ := set $_collectorServiceTLS "key" "@?secrets/collector-key.pem" }}
{{ else }}
  {{ $_ := set $_collectorServiceTLS "_cert" "" }}
  {{ $_ := set $_collectorServiceTLS "_key" "" }}
{{ end }}
{{ $_ := set $_collector "image" $_collectorImage }}
{{ $_ := set $_collector "resources" $_values.config.collectorResources }}
{{ $_ := set $_collector "complianceResources" $_values.config.complianceResources }}
{{ $_ := set $_collector "collectionMethod" $_values.config.collectionMethod }}
{{ $_ := set $_collector "disableTaintTolerations" $_values.config.disableTaintTolerations }}
{{ $_ := set $_collector "slimMode" $_values.config.slimCollector }}
{{ $_ := set $_collector "serviceTLS" $_collectorServiceTLS }}
{{ $_ := set $_collector "exposeMonitoring" $_values.config.exposeMonitoring }}

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
{{ $_ := set $newValues "mainImagePullSecrets" $_mainImagePullSecrets }}
{{ $_ := set $newValues "collectorImagePullSecrets" $_collectorImagePullSecrets }}
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
{{ end }}

{{/*
    $rox / ._rox is the dictionary in which _all_ data that is modified by the init logic
    is stored.
    We ensure that it has the required shape, and then right after merging the user-specified
    $.Values, we apply some bootstrap defaults.
   */}}
{{ $rox := deepCopy $.Values }}
{{ $_ := include "srox.mergeInto" (list $rox ($.Files.Get "internal/config-shape.yaml" | fromYaml) ($.Files.Get "internal/bootstrap-defaults.yaml" | fromYaml)) }}
{{ $_ = set $ "_rox" $rox }}

{{/* Set the config fingerprint */}}
{{ $_ = set $._rox "_configFP" $configFP }}

{{/* Global state (accessed from sub-templates) */}}
{{ $state := dict "notes" list "warnings" list "referencedImages" dict }}
{{ $_ = set $._rox "_state" $state }}

{{/*
    General validation (before merging in defaults).
   */}}

{{ if kindIs "invalid" .Values.initToken }}
  {{ include "srox.note" (list $ "No cluster init token specified using the parameter 'initToken', executing chart in compatibility mode.") }}
{{ end }}

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

{{ if and $._rox.admissionControl.dynamic.enforceOnUpdates (not $._rox.admissionControl.listenOnUpdates) }}
  {{ include "srox.warn" (list $ "Incompatible settings: 'admissionControl.dynamic.enforceOnUpdates' is set to true, while `admissionControl.listenOnUpdates` is set to false. For the feature to be active, enable both settings by setting them to true.") }}
{{ end }}

{{ end }}

{{ if and $._rox.collector.slimMode (not (kindIs "invalid" $._rox.collector.image.tag)) }}
  {{ $msg := "You have enabled slimMode for collecter and overriden the collector image tag. Make sure that the referenced image is a slim collector image." }}
  {{ include "srox.warn" (list $ $msg) }}
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
{{ $_ = include "srox.mergeInto" (list $defaultsCfg (fromYaml $platformCfgFile.contents) (tpl ($.Files.Get "internal/defaults.yaml") . | fromYaml)) }}
{{ $_ = set $rox "_defaults" $defaultsCfg }}
{{ $_ = include "srox.mergeInto" (list $rox $defaultsCfg.defaults) }}


{{/* Expand applicable config values */}}
{{ $expandables := $.Files.Get "internal/expandables.yaml" | fromYaml }}
{{ include "srox.expandAll" (list $ $rox $expandables) }}


{{/* Initial image pull secret setup. */}}
{{ include "srox.configureImagePullSecrets" (list $ "mainImagePullSecrets" $._rox.mainImagePullSecrets (list "stackrox")) }}
{{ include "srox.configureImagePullSecrets" (list $ "collectorImagePullSecrets" $._rox.collectorImagePullSecrets (list "stackrox" "collector-stackrox")) }}

{{/*
    Final validation (after merging in defaults).
   */}}

{{ if eq ._rox.clusterName "" }}
  {{ if $_compatibilityMode }}
    {{ include "srox.fail" "No cluster name specified. Set 'cluster.name' to the desired cluster name." }}
  {{ else }}
    {{ include "srox.fail" "No cluster name specified. Set 'clusterName' to the desired cluster name." }}
  {{ end }}
{{ end}}

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

{{ include "srox.configureImagePullSecretsForDockerRegistry" (list $ ._rox.mainImagePullSecrets) }}
{{ include "srox.configureImagePullSecretsForDockerRegistry" (list $ ._rox.collectorImagePullSecrets) }}

{{ end }}

{{ end }}

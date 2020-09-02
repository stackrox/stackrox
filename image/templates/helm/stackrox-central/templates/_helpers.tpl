{{- define "stackrox.init" -}}

{{/*
    This template sets up the _rox structure, containing everything required by the resource template files.
   */}}

{{- if not ._rox -}}

{{/*
    Initial Setup
   */}}
{{- $warnings := (list) -}}
{{- $notes := (list) -}}
{{- $customCertGen := false -}}
{{- $adminPasswordGenerated := false -}}
{{- $mainImageTag := default .Chart.AppVersion .Values.mainImageTag -}}
{{- $mainImageRepository := default "stackrox.io/main" .Values.mainImageRepository -}}


{{/*
    Normalization.
  */}}
{{- if not .Values.central -}}{{- $_ := set .Values "central" dict -}}{{- end -}}
{{- if not .Values.persistence -}}{{- $_ := set .Values "persistence" (dict "pv" dict) -}}{{- end -}}

{{/*
     Generate TLS Certificates.
   */}}
{{- $caCert := genCA "StackRox Certificate Authority" 1825 -}}
{{- $centralCN := "central.stackrox" -}}
{{- $centralSANs := list $centralCN (printf "%s.svc" $centralCN) -}}
{{- $centralCert := genSignedCert $centralCN nil $centralSANs 365 $caCert -}}
{{- $customCertGen = true -}}

{{/*
    Generate Admin Password.
   */}}
{{- $adminPassword := "" -}}
{{- if not .Values.central.adminPassword -}}
  {{- $adminPassword = randAlphaNum 32 -}}
  {{- $adminPasswordGenerated = true -}}
{{- else -}}
  {{- $adminPassword = .Values.central.adminPassword -}}
{{- end -}}

{{/*
     Generate JWT Key.
  */}}
{{- $jwtKey := genPrivateKey "rsa" -}}

{{/*
    Setup Default TLS Certificate.
   */}}
{{- $defaultTlsCert := dict -}}
{{- if .Values.defaultTlsCert -}}
  {{- $_ := set $defaultTlsCert "cert" (required "defaultTlsCert.cert must be provided" .Values.defaultTlsCert.cert) -}}
  {{- $_ := set $defaultTlsCert "key" (required "defaultTlsCert.key must be provided" .Values.defaultTlsCert.key) -}}
  {{- $notes = append $notes "Configured default TLS certificate" -}}
{{- end -}}

{{/*
    Setup configuration for persistence backend.
  */}}
{{- $persistenceConf := dict -}}
{{- if .Values.persistence -}}
  {{/*
       Sanity checks for user provided persistence configuration.
     */}}
  {{- if and (hasKey .Values.persistence "pv") (hasKey .Values.persistence "hostpath") -}}
    {{- fail "Multiple persistence backends selected" -}}
  {{- end -}}

  {{- if hasKey .Values.persistence "pv" -}}
    {{/*
         Handle Persistent Volumes
       */}}
    {{- $persistenceConf = mustMergeOverwrite .Values.defaults.persistence.pv .Values.persistence.pv -}}
    {{- $notes = append $notes (printf "Using persistent volume (size: %v)" $persistenceConf.size) -}}
  {{- else if hasKey .Values.persistence "hostpath" -}}
    {{/*
         Handle HostPath
       */}}
      {{- $persistenceConf = mustMergeOverwrite .Values.defaults.persistence.hostpath (dict "value" (dict "hostPath" (dict "path" .Values.persistence.hostpath))) -}}
      {{- $notes = append $notes (printf "Using host path '%s' for persistence" ($persistenceConf.value.hostPath.path)) -}}
  {{- else -}}
    {{- fail "Invalid persistence configuration" -}}
  {{- end -}}
{{- else -}}
  {{/*
       Setup default persistence.
     */}}
  {{- $notes = append $notes "Using default persistent backend 'pv' (Persistent Volume)" -}}
  {{- $persistenceConf = .Values.defaults.persistence.pv -}}
{{- end -}}

{{/*
    Setup Environment Variables.
   */}}
{{- $env := dict -}}
{{- $_ := set $env "telemetryEnabled" (default "true" .Values.telemetryEnabled) -}}
{{- $_ := set $env "offlineMode" (default "false" .Values.offlineMode) -}}

{{/*
    Environment Feature Detection.
   */}}
{{- $openShiftCluster := false -}}
{{- if .Capabilities.APIVersions.Has "apps.openshift.io/v1" -}}
  {{- $openShiftCluster = true -}}
{{- end -}}

{{/*
    Setup Image Pull Secrets for Docker Registry.
   */}}
{{- $dockerConfig := dict -}}
{{- if .Values.imagePullSecret -}}
{{- $username := .Values.imagePullSecret.username -}}
{{- $password := .Values.imagePullSecret.password -}}
{{- $registry := default "https://stackrox.io" .Values.imagePullSecret.registry -}}
{{- $auth := dict "auth" (printf "%s:%s" $username $password | b64enc) -}}
{{- $registryAuth := dict $registry $auth -}}
{{- $dockerAuths := dict "auths" $registryAuth -}}
{{- $_ := set $dockerConfig "registry" $registry -}}
{{- $_ := set $dockerConfig "auths" $dockerAuths -}}
{{- else -}}
  {{- $warnings = append $warnings "No Image Pull Secrets provided. Make sure they are set up properly." -}}
{{- end -}}

{{/*
    Setup License.
   */}}
{{- $license := "" -}}
{{- if .Values.license -}}
  {{- $license = .Values.license -}}
{{- else -}}
  {{- $warnings = append $warnings "No StackRox license provided. Make sure a valid license exists in Kubernetes secret 'central-license'." -}}
{{- end -}}

{{/*
    Assemble Global Configuration.
   */}}
{{- $globalCfg := dict -}}
{{- $_ := set $globalCfg "caCert" $caCert -}}
{{- $_ := set $globalCfg "persistence" $persistenceConf -}}
{{- $_ := set $globalCfg "docker" $dockerConfig -}}
{{- $_ := set $globalCfg "license" $license -}}
{{- $_ := set $globalCfg "openShiftCluster" $openShiftCluster -}}

{{/*
    Assemble Central Configuration.
   */}}
{{- $centralCfg := dict -}}
{{- $_ := set $centralCfg "tlsCert" $centralCert -}}
{{- $_ := set $centralCfg "jwtKey" $jwtKey -}}
{{- $_ := set $centralCfg "env" $env -}}
{{- $_ := set $centralCfg "endpoints" (.Files.Get "config/endpoints.yaml.default") -}}
{{- $_ := set $centralCfg "mainImageTag" $mainImageTag -}}
{{- $_ := set $centralCfg "mainImageRepository" $mainImageRepository -}}

{{- $_ := set $centralCfg "defaultTlsCert" $defaultTlsCert -}}
{{- $_ := set $centralCfg "resources" .Values.centralResources -}}
{{- $_ := set $centralCfg "adminPassword" $adminPassword -}}
{{- $_ := set $centralCfg "adminPasswordGenerated" $adminPasswordGenerated -}}

{{- $_ := set $centralCfg "nodeSelector" .Values.central.nodeSelector -}}

{{- if $customCertGen -}}
  {{- $warnings = append $warnings "Helm has generated at least one TLS certificate. For compatibility reasons, this certificate uses a 4096-bit RSA key. For improved performance and security, consider generating certificates with elliptic curve (ECDSA) keys using `roxctl foo bar baz`." -}}
{{- end -}}

{{- $warnings = append $warnings "This helm chart is still experimental. Use with caution." -}}

{{/*
    Assemble _rox Value.
   */}}
{{- $rox := dict -}}
{{- $_ := set $rox "version" .Chart.AppVersion -}}
{{- $_ := set $rox "global" $globalCfg -}}
{{- $_ := set $rox "central" $centralCfg -}}
{{- $_ := set $rox "warnings" $warnings -}}
{{- $_ := set $rox "notes" $notes -}}

{{- $_ := set . "_rox" $rox -}}
{{- end -}}
{{- end -}}

{{/*
    Specialized Template Functions to be included from the Template Files.
   */}}

{{- define "defaultLabels" -}}
    app.kubernetes.io/name: stackrox
    app.kubernetes.io/managed-by: Helm
{{- end }}

{{- define "defaultAnnotations" -}}
    meta.helm.sh/release-namespace: stackrox
    meta.helm.sh/release-name: {{ .Release.Name }}
{{- end }}

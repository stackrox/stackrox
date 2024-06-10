{{/*
  srox.scannerV4Volume $

  Configures and initializes Scanner v4 persistence.
   */}}
{{ define "srox.scannerV4Volume" }}
{{ $ := . }}
{{ $_ := set $ "_rox" $._rox }}

{{ $scannerV4DBCfg := $._rox.scannerV4.db }}

{{/*
    Scanner v4 DB Volume config setup.
  */}}
{{ $scannerV4DBVolumeCfg := dict }}
{{ $scannerV4DBVolumeHumanReadable := "" }}
{{ $pvcConfigShape := $._rox._configShape.scannerV4.db.persistence.persistentVolumeClaim }}
{{ $pvcDefaults := dict }}
{{- if eq $.Chart.Name "stackrox-central-services" -}}
  {{ $pvcDefaults = $._rox._defaults.scannerV4DBPVCDefaults }}
{{- else -}}
  {{ $pvcDefaults = $._rox.scannerV4DBPVCDefaults }}
{{- end -}}

{{ $extraSettings := dict "createClaim" .Release.IsInstall }}
{{ if $._rox.env.storageClasses.default }}
  {{ $_ = set $extraSettings "storageClass" $._rox.env.storageClasses.default }}
{{ end }}

{{/* First we check that the persistence configuration provided by the user is sane in the sense that only one of the
     supported backends emptyDir/hostPath/PVC is configured. */}}
{{ $persistenceBackendsConfigured := list }}
{{ if $scannerV4DBCfg.persistence.none }}
  {{ $persistenceBackendsConfigured = append $persistenceBackendsConfigured "emptyDir" }}
{{ end }}
{{ if $scannerV4DBCfg.persistence.hostPath }}
  {{ $persistenceBackendsConfigured = append $persistenceBackendsConfigured "hostPath" }}
{{ end }}
{{ if not (deepEqual $pvcConfigShape $scannerV4DBCfg.persistence.persistentVolumeClaim) }}
  {{ $persistenceBackendsConfigured = append $persistenceBackendsConfigured (printf "PVC:%v" $scannerV4DBCfg.persistence.persistentVolumeClaim) }}
{{ end }}

{{/* Sanity checks and defaulting. */}}
{{ if empty $persistenceBackendsConfigured }}
  {{/* No persistence backend configured, pick a reasonable default. */}}
  {{ if or $._rox.env.storageClasses.default (eq $.Chart.Name "stackrox-central-services") }}
    {{/* Either a default StorageClass has been detected or we are currently rendering central-services.
         In both cases we configure a PVC as persistence backend. */}}
    {{ if $._rox.env.storageClasses.default }}
      {{ include "srox.note" (list $ "Default StorageClass detected, a PVC will be used for Scanner V4 DB persistence.") }}
    {{ else }}
      {{ include "srox.note" (list $ "A PVC will be used for Scanner V4 DB persistence.") }}
    {{ end }}
    {{ if has $pvcDefaults.claimName $._rox.env.pvcs.names }}
      {{ $_ = set $extraSettings "createClaim" false }}
    {{ else }}
      {{ $_ = set $extraSettings "createClaim" true }}
    {{ end }}
    {{ $_ = include "srox.mergeInto" (list $scannerV4DBCfg.persistence.persistentVolumeClaim $pvcConfigShape $extraSettings $pvcDefaults) }}
  {{ else }}
    {{/* No default StorageClass detected, currently rendering secured-cluster-services chart. */}}
    {{ if has $pvcDefaults.claimName $._rox.env.pvcs.names }}
      {{ include "srox.note" (list $ (printf "A PVC named %s already exists, will keep using it for Scanner V4 DB persistence." $pvcDefaults.claimName)) }}
      {{ $_ = set $extraSettings "createClaim" false }}
      {{ $_ = include "srox.mergeInto" (list $scannerV4DBCfg.persistence.persistentVolumeClaim $pvcConfigShape $extraSettings $pvcDefaults) }}
    {{ else }}
      {{/* Fallback to emptyDir. */}}
      {{ include "srox.warn" (list $ (printf "No default StorageClass detected, using emptyDir as persistence backend for Scanner V4 DB. It is highly recommended to use a PVC instead. Please check the documentation for more information on this." )) }}
      {{ $_ = set $scannerV4DBCfg.persistence "none" true }}
    {{ end }}
  {{ end }}
{{ else if gt (len $persistenceBackendsConfigured) 1 }}
  {{ include "srox.fail" (printf "Invalid persistence configuration for Scanner V4 DB: more than one persistence backend configured (%v)" $persistenceBackendsConfigured) }}
{{ end }}


{{/* Update $scannerV4DBVolumeCfg depending on configured persistence backend. */}}
{{ if $scannerV4DBCfg.persistence.none }}
  {{ include "srox.warn" (list $ "Persistence for Scanner V4 DB is turned off (it is using an emptyDir volume). Every deletion of the StackRox Scanner V4 DB pod will cause you to lose all your data. This is STRONGLY recommended against.") }}
  {{ $_ := set $scannerV4DBVolumeCfg "emptyDir" dict }}
  {{ $scannerV4DBVolumeHumanReadable = "emptyDir" }}
{{ else if $scannerV4DBCfg.persistence.hostPath }}
  {{ if not $scannerV4DBCfg.nodeSelector }}
    {{ include "srox.warn" (list $ "A hostPath volume will be used by the Scanner V4 DB. At the same time no node selector is specified. This is unlikely to work reliably.") }}
  {{ end }}
  {{ $_ := set $scannerV4DBVolumeCfg "hostPath" (dict "path" $scannerV4DBCfg.persistence.hostPath) }}
  {{ $scannerV4DBVolumeHumanReadable = printf "hostPath (%s)" $scannerV4DBCfg.persistence.hostPath}}
{{ else }}
  {{ if kindIs "invalid" $scannerV4DBCfg.persistence.persistentVolumeClaim.storageClass }}
    {{ include "srox.note" (list $ "A PVC using the default storage class will be used for the Scanner V4 DB.") }}
  {{ else }}
    {{ include "srox.note" (list $ (printf "A PVC using the storage class %q will be used for the Scanner V4 DB." $scannerV4DBCfg.persistence.persistentVolumeClaim.storageClass)) }}
  {{ end }}
  {{ $scannerV4DBPVCCfg := $scannerV4DBCfg.persistence.persistentVolumeClaim }}
  {{ $_ := include "srox.mergeInto" (list $scannerV4DBPVCCfg $extraSettings $pvcDefaults) }}
  {{ $_ = set $scannerV4DBVolumeCfg "persistentVolumeClaim" (dict "claimName" $scannerV4DBPVCCfg.claimName) }}
  {{ if $scannerV4DBPVCCfg.createClaim }}
    {{ $_ = set $scannerV4DBCfg.persistence "_pvcCfg" $scannerV4DBPVCCfg }}
  {{ end }}
  {{ if $scannerV4DBPVCCfg.storageClass }}
    {{ $_ = set $._rox._state "referencedStorageClasses" (mustAppend $._rox._state.referencedStorageClasses $scannerV4DBPVCCfg.storageClass | uniq) }}
  {{ end }}
  {{ $scannerV4DBVolumeHumanReadable = printf "PVC (%s)" $scannerV4DBPVCCfg.claimName }}
{{ end }}

{{ $allPersistenceMethods := keys $scannerV4DBVolumeCfg | sortAlpha }}
{{ if ne (len $allPersistenceMethods) 1 }}
{{ end }}

{{ $_ = set $scannerV4DBCfg.persistence "_volumeCfg" $scannerV4DBVolumeCfg }}
{{ $_ := set $._rox "_scannerV4Volume" $scannerV4DBVolumeHumanReadable }}

{{ end }}

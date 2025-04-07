{{/*
  srox.centralSetup $

  Configures and initializes central specific values like certificates, admin password or persistence.
   */}}
{{ define "srox.centralSetup" }}
{{ $ := . }}
{{ $env := $._rox.env }}
{{ $_ := set $ "_rox" $._rox }}
{{ $centralCfg := $._rox.central }}
{{ $centralDBCfg := $._rox.central.db }}

{{/* Image settings */}}
{{ include "srox.configureImage" (list $ $centralCfg.image) }}

{{/* Admin password */}}
{{ include "srox.configurePassword" (list $ "central.adminPassword" "admin") }}

{{/* Service TLS Certificates */}}
{{ $centralCertSpec := dict "CN" "CENTRAL_SERVICE: Central" "dnsBase" "central" }}
{{ include "srox.configureCrypto" (list $ "central.serviceTLS" $centralCertSpec) }}

{{/* JWT Token Signer */}}
{{ $jwtSignerSpec := dict "keyOnly" "rsa" }}
{{ include "srox.configureCrypto" (list $ "central.jwtSigner" $jwtSignerSpec) }}

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

{{/* Central DB password */}}
{{/* Always set up the password for Postgres if it is enabled */}}
{{ include "srox.configurePassword" (list $ "central.db.password") }}
{{ if not $centralDBCfg.external }}
{{ include "srox.configureImage" (list $ $centralDBCfg.image) }}

{{/* Central DB Service TLS Certificates */}}
{{ $centralDBCertSpec := dict "CN" "CENTRAL_DB_SERVICE: Central DB" "dnsBase" "central-db" }}
{{ include "srox.configureCrypto" (list $ "central.db.serviceTLS" $centralDBCertSpec) }}
{{ end }}

{{/*
    Central's DB PVC config setup
  */}}
{{ $dbVolumeCfg := dict }}
{{ $dbBackupVolumeCfg := dict }}
{{ if not $centralDBCfg.external }}
{{ if $centralDBCfg.persistence.none }}
  {{ include "srox.warn" (list $ "You have selected no persistence backend. Every deletion of the StackRox Central DB pod will cause you to lose all your data. This is STRONGLY recommended against.") }}
  {{ $_ := set $dbVolumeCfg "emptyDir" dict }}
  {{ $_ := set $dbBackupVolumeCfg "emptyDir" dict }}
{{ end }}
{{ if $centralDBCfg.persistence.hostPath }}
  {{ if not $centralDBCfg.nodeSelector }}
    {{ include "srox.warn" (list $ "You have selected host path persistence, but not specified a node selector. This is unlikely to work reliably.") }}
  {{ end }}
  {{ $_ := set $dbVolumeCfg "hostPath" (dict "path" $centralDBCfg.persistence.hostPath) }}
  {{ $_ := set $dbBackupVolumeCfg "hostPath" (printf "%s/backup" (dict "path" $centralDBCfg.persistence.hostPath)) }}
{{ end }}
{{/* Configure PVC if either any of the settings in `centralDB.persistence.persistentVolumeClaim` are provided,
     or no other persistence backend has been configured yet. */}}
{{ if or (not (deepEqual $._rox._configShape.central.db.persistence.persistentVolumeClaim $centralDBCfg.persistence.persistentVolumeClaim)) (not $dbVolumeCfg) }}
  {{ $dbPVCCfg := $centralDBCfg.persistence.persistentVolumeClaim }}
  {{ $dbBackupPVCCfg := $centralDBCfg.persistence.persistentVolumeClaim }}
  {{ $_ := include "srox.mergeInto" (list $dbPVCCfg $._rox._defaults.dbPVCDefaults (dict "createClaim" (or .Release.IsInstall (eq $._rox._renderMode "centralDBOnly")))) }}
  {{ $_ := include "srox.mergeInto" (list $dbBackupPVCCfg $._rox._defaults.dbPVCDefaults (dict "createClaim" (or .Release.IsInstall (eq $._rox._renderMode "centralDBOnly") $centralDBCfg.persistence.createBackup ))) }}
  {{ $_ = set $dbVolumeCfg "persistentVolumeClaim" (dict "claimName" $dbPVCCfg.claimName) }}
  {{ $_ = set $dbBackupVolumeCfg "persistentVolumeClaim" (dict "claimName" (printf "%s-backup" $dbPVCCfg.claimName)) }}
  {{ if $dbPVCCfg.createClaim }}
    {{ $_ = set $centralDBCfg.persistence "_pvcCfg" $dbPVCCfg }}
  {{ end }}
  {{ if $dbBackupPVCCfg.createClaim }}
    {{ $_ = set $centralDBCfg.persistence "_backupPVCCfg" $dbBackupPVCCfg }}
  {{ end }}
  {{ if $dbPVCCfg.storageClass}}
    {{ $_ = set $._rox._state "referencedStorageClasses" (mustAppend $._rox._state.referencedStorageClasses $dbPVCCfg.storageClass | uniq) }}
  {{ end }}
{{ end }}
{{ end }}

{{ if not $centralDBCfg.external }}
{{ $_ = set $centralDBCfg.persistence "_volumeCfg" $dbVolumeCfg }}
{{ $_ = set $centralDBCfg.persistence "_backupVolumeCfg" $dbBackupVolumeCfg }}
{{ end }}

{{/* Endpoint configuration */}}
{{ include "srox.configureCentralEndpoints" $._rox.central }}

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
{{ end }}

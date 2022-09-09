{{/*
  srox.centralSetup $

  Configures and initializes central specific values like certificates, admin password or persistence.
   */}}
{{ define "srox.centralSetup" }}
{{ $ := . }}
{{ $env := $._rox.env }}
{{ $_ := set $ "_rox" $._rox }}
{{ $centralCfg := $._rox.central }}
{{ $centralDBCfg := $._rox.centralDB }}

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
{{ if $centralDBCfg.enabled }}
{{/* Always set up the password for Postgres if it is enabled */}}
{{ include "srox.configurePassword" (list $ "centralDB.password") }}
{{ if not $centralDBCfg.preexisting }}
{{ include "srox.configureImage" (list $ $centralDBCfg.image) }}

{{/* Central DB Service TLS Certificates */}}
{{ $centralDBCertSpec := dict "CN" "CENTRAL_DB_SERVICE: Central DB" "dnsBase" "central-db" }}
{{ include "srox.configureCrypto" (list $ "centralDB.serviceTLS" $centralDBCertSpec) }}
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
{{/* Configure PVC if either any of the settings in `central.persistence.persistentVolumeClaim` are provided,
     or no other persistence backend has been configured yet. */}}
{{ if or (not (deepEqual $._rox._configShape.central.persistence.persistentVolumeClaim $centralCfg.persistence.persistentVolumeClaim)) (not $volumeCfg) }}
  {{ $pvcCfg := $centralCfg.persistence.persistentVolumeClaim }}
  {{ $_ := include "srox.mergeInto" (list $pvcCfg $._rox._defaults.pvcDefaults (dict "createClaim" $.Release.IsInstall)) }}
  {{ $_ = set $volumeCfg "persistentVolumeClaim" (dict "claimName" $pvcCfg.claimName) }}
  {{ if $pvcCfg.createClaim }}
    {{ $_ = set $centralCfg.persistence "_pvcCfg" $pvcCfg }}
  {{ end }}
{{ end }}

{{ $dbVolumeCfg := dict }}
{{ if and $centralDBCfg.enabled (not $centralDBCfg.preexisting) }}
{{ if $centralDBCfg.persistence.none }}
  {{ include "srox.warn" (list $ "You have selected no persistence backend. Every deletion of the StackRox Central DB pod will cause you to lose all your data. This is STRONGLY recommended against.") }}
  {{ $_ := set $dbVolumeCfg "emptyDir" dict }}
{{ end }}
{{ if $centralDBCfg.persistence.hostPath }}
  {{ if not $centralDBCfg.nodeSelector }}
    {{ include "srox.warn" (list $ "You have selected host path persistence, but not specified a node selector. This is unlikely to work reliably.") }}
  {{ end }}
  {{ $_ := set $dbVolumeCfg "hostPath" (dict "path" $centralDBCfg.persistence.hostPath) }}
{{ end }}
{{/* Configure PVC if either any of the settings in `centralDB.persistence.persistentVolumeClaim` are provided,
     or no other persistence backend has been configured yet. */}}
{{ if or (not (deepEqual $._rox._configShape.centralDB.persistence.persistentVolumeClaim $centralDBCfg.persistence.persistentVolumeClaim)) (not $dbVolumeCfg) }}
  {{ $dbPvcCfg := $centralDBCfg.persistence.persistentVolumeClaim }}
  {{ $_ := include "srox.mergeInto" (list $dbPvcCfg $._rox._defaults.dbPVCDefaults (dict "createClaim" $.Release.IsInstall)) }}
  {{ $_ = set $dbVolumeCfg "persistentVolumeClaim" (dict "claimName" $dbPvcCfg.claimName) }}
  {{ if $dbPvcCfg.createClaim }}
    {{ $_ = set $centralDBCfg.persistence "_pvcCfg" $dbPvcCfg }}
  {{ end }}
{{ end }}
{{ end }}

{{ $allPersistenceMethods := keys $volumeCfg | sortAlpha }}
{{ if ne (len $allPersistenceMethods) 1 }}
  {{ include "srox.fail" (printf "Invalid or no persistence configurations for central: [%s]" (join "," $allPersistenceMethods)) }}
{{ end }}
{{ $_ = set $centralCfg.persistence "_volumeCfg" $volumeCfg }}
{{ if and $centralDBCfg.enabled (not $centralDBCfg.preexisting) }}
{{ $_ = set $centralDBCfg.persistence "_volumeCfg" $dbVolumeCfg }}
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

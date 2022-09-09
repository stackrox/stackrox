package renderer

import (
	"text/template"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/features"
	helmTemplate "github.com/stackrox/rox/pkg/helm/template"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/zip"
)

const (
	publicValuesTemplateStr = `
# StackRox Central Services Chart - PUBLIC configuration values.
#
# These are the public values for the deployment of the StackRox Central Services chart.
# You can safely store this file in a source-code management system, as it does not contain
# sensitive information.
# It is recommended to reference this file via the '-f' option whenever running a 'helm upgrade'
# command.

env:
  {{- if eq .ClusterType.String "OPENSHIFT_CLUSTER" }}
  openshift: 3
  {{- else if eq .ClusterType.String "OPENSHIFT4_CLUSTER" }}
  openshift: 4
  {{- end }}
  offlineMode: {{ .K8sConfig.OfflineMode }}
  {{- if ne .K8sConfig.IstioVersion "" }}
  istio: true
  {{- end }}

imagePullSecrets:
  useExisting:
  - stackrox
  {{- if and .K8sConfig.ScannerSecretName (ne .K8sConfig.ScannerSecretName "stackrox") }}
  - {{ .K8sConfig.ScannerSecretName | quote }}
  {{- end }}

{{- if .K8sConfig.ImageOverrides.MainRegistry }}
image:
  registry: {{ .K8sConfig.ImageOverrides.MainRegistry }}
{{- end }}

central:
  disableTelemetry: {{ not .K8sConfig.EnableTelemetry }}

  {{- if ne (.GetConfigOverride "endpoints.yaml") "" }}
  endpointsConfig: |
    {{- .GetConfigOverride "endpoints.yaml" | nindent 4 }}
  {{- end }}

  {{- if .HostPath }}
  {{- if .HostPath.WithNodeSelector }}
  nodeSelector:
    {{ .HostPath.NodeSelectorKey | quote }}: {{ .HostPath.NodeSelectorValue | quote }}
  {{- end }}
  {{- end }}

  {{- if .K8sConfig.ImageOverrides.Main }}
  image:
    {{- if .K8sConfig.ImageOverrides.Main.Name }}
    name: {{ .K8sConfig.ImageOverrides.Main.Name }}
    {{- end }}
    {{- if .K8sConfig.ImageOverrides.Main.Tag }}
    # WARNING: You are using a non-default main image tag. Upgrades via 'helm upgrade'
    # will not work as expected. To ensure a smooth upgrade experience, make sure
    # StackRox images are mirrored with the same tags as in the stackrox.io registry.
    tag: {{ .K8sConfig.ImageOverrides.Main.Tag }}
    {{- end }}
  {{- end }}
  {{- if .K8sConfig.ImageOverrides.CentralDB }}
  dbImage:
    {{- if .K8sConfig.ImageOverrides.CentralDB.Registry }}
    registry: {{ .K8sConfig.ImageOverrides.CentralDB.Registry }}
    {{- end }}
    {{- if .K8sConfig.ImageOverrides.CentralDB.Name }}
    name: {{ .K8sConfig.ImageOverrides.CentralDB.Name }}
    {{- end }}
    {{- if .K8sConfig.ImageOverrides.CentralDB.Tag }}
    # WARNING: You are using a non-default Central DB image tag. Upgrades via
    # 'helm upgrade' will not work as expected. To ensure a smooth upgrade experience,
    # make sure StackRox images are mirrored with the same tags as in the stackrox.io
    # registry.
    tag: {{ .K8sConfig.ImageOverrides.CentralDB.Tag }}
    {{- end }}
  {{- end }}
  persistence:
    {{- if .HostPath }}
    hostPath: {{ .HostPath.HostPath }}
    {{ else if .External }}
    persistentVolumeClaim:
      claimName: {{ .External.Name | quote }}
      size: {{ printf "%dGi" .External.Size | quote }}
      {{- if .External.StorageClass }}
      storageClass: {{ .External.StorageClass | quote }}
      {{- end }}
    {{- else }}
    none: true
    {{- end }}

  {{- if ne .K8sConfig.LoadBalancerType.String "NONE" }}
  exposure:
    {{- if eq .K8sConfig.LoadBalancerType.String "LOAD_BALANCER" }}
    loadBalancer:
      enabled: true
      port: 443
    {{ else if eq .K8sConfig.LoadBalancerType.String "NODE_PORT" }}
    nodePort:
      enabled: true
    {{ else if eq .K8sConfig.LoadBalancerType.String "ROUTE" }}
    route:
      enabled: true
    {{ end }}
  {{- end }}

centralDB:
  enabled: {{ .K8sConfig.EnableCentralDB }}
  {{- if .HostPath }}
  {{- if .HostPath.WithNodeSelector }}
  nodeSelector:
    {{ .HostPath.NodeSelectorKey | quote }}: {{ .HostPath.NodeSelectorValue | quote }}
  {{- end }}
  {{- end }}

  {{- if .K8sConfig.ImageOverrides.CentralDB }}
  image:
    {{- if .K8sConfig.ImageOverrides.CentralDB.Registry }}
    registry: {{ .K8sConfig.ImageOverrides.CentralDB.Registry }}
    {{- end }}
    {{- if .K8sConfig.ImageOverrides.CentralDB.Name }}
    name: {{ .K8sConfig.ImageOverrides.CentralDB.Name }}
    {{- end }}
    {{- if .K8sConfig.ImageOverrides.CentralDB.Tag }}
    # WARNING: You are using a non-default Central DB image tag. Upgrades via
    # 'helm upgrade' will not work as expected. To ensure a smooth upgrade experience,
    # make sure StackRox images are mirrored with the same tags as in the stackrox.io
    # registry.
    tag: {{ .K8sConfig.ImageOverrides.CentralDB.Tag }}
    {{- end }}
  {{- end }}
  persistence:
    {{- if .HostPath }}
    hostPath: {{ .HostPath.HostPath }}
    {{ else if .External }}
    persistentVolumeClaim:
      claimName: {{ .External.Name | quote }}
      size: {{ printf "%dGi" .External.Size | quote }}
      {{- if .External.StorageClass }}
      storageClass: {{ .External.StorageClass | quote }}
      {{- end }}
    {{- else }}
    none: true
    {{- end }}

scanner:
  # IMPORTANT: If you do not wish to run StackRox Scanner, change the value on the following
  # line to "true".
  disable: false

  {{- if .K8sConfig.ImageOverrides.Scanner }}
  image:
    {{- if .K8sConfig.ImageOverrides.Scanner.Registry }}
    registry: {{ .K8sConfig.ImageOverrides.Scanner.Registry }}
    {{- end }}
    {{- if .K8sConfig.ImageOverrides.Scanner.Name }}
    name: {{ .K8sConfig.ImageOverrides.Scanner.Name }}
    {{- end }}
    {{- if .K8sConfig.ImageOverrides.Scanner.Tag }}
    # WARNING: You are using a non-default Scanner image tag. Upgrades via 'helm upgrade'
    # will not work as expected. To ensure a smooth upgrade experience, make sure
    # StackRox images are mirrored with the same tags as in the stackrox.io registry.
    tag: {{ .K8sConfig.ImageOverrides.Scanner.Tag }}
    {{- end }}
  {{- end }}

  {{- if .K8sConfig.ImageOverrides.ScannerDB }}
  dbImage:
    {{- if .K8sConfig.ImageOverrides.ScannerDB.Registry }}
    registry: {{ .K8sConfig.ImageOverrides.ScannerDB.Registry }}
    {{- end }}
    {{- if .K8sConfig.ImageOverrides.ScannerDB.Name }}
    name: {{ .K8sConfig.ImageOverrides.ScannerDB.Name }}
    {{- end }}
    {{- if .K8sConfig.ImageOverrides.ScannerDB.Tag }}
    # WARNING: You are using a non-default Scanner DB image tag. Upgrades via
    # 'helm upgrade' will not work as expected. To ensure a smooth upgrade experience,
    # make sure StackRox images are mirrored with the same tags as in the stackrox.io
    # registry.
    tag: {{ .K8sConfig.ImageOverrides.ScannerDB.Tag }}
    {{- end }}
  {{- end }}

{{- $envVars := deepCopy .EnvironmentMap -}}
{{- $_ := unset $envVars "ROX_OFFLINE_MODE" -}}
{{- $_ := unset $envVars "ROX_INIT_TELEMETRY_ENABLED" -}}
{{- if $envVars }}

customize:
  # Custom environment variables that will be applied to all containers
  # of all workloads.
  envVars:
    {{ range $key, $value := $envVars -}}
    {{ quote $key }}: {{ quote $value }}
    {{ end }}
{{- end }}
`
	privateValuesYamlTemplateStr = `
# StackRox Central Services chart - SECRET configuration values.
#
# These are secret values for the deployment of the StackRox Central Services chart.
# Store this file in a safe place, such as a secrets management system.
# Note that these values are usually NOT required when upgrading or applying configuration
# changes, but they are required for re-deploying an exact copy to a separate cluster.

{{- if ne (index .SecretsBase64Map "ca.pem") "" }}
# Internal service TLS Certificate Authority
ca:
  cert: |
    {{- index .SecretsBase64Map "ca.pem" | b64dec | nindent 4 }}
  key: |
    {{- index .SecretsBase64Map "ca-key.pem" | b64dec | nindent 4 }}
{{- end }}

{{- if ne (index .SecretsBase64Map "central-license") "" }}
# StackRox license key
licenseKey: | 
  {{- index .SecretsBase64Map "central-license" | b64dec | nindent 2 }}
{{- end }}

# Configuration secrets for the Central deployment
central:
  {{- if ne (index .SecretsBase64Map "htpasswd") "" }}
  # Administrator password for logging in to the StackRox Portal.
  # htpasswd (bcrypt) encoded for security reasons, consult the "password" file
  # that is part of the deployment bundle for the raw password.
  adminPassword:
    htpasswd: |
      {{- index .SecretsBase64Map "htpasswd" | b64dec | nindent 6 }}
  {{- end }}

  {{- if ne (index .SecretsBase64Map "jwt-key.pem") "" }}
  # Private key used for signing JWT tokens.
  jwtSigner:
    key: |
      {{- index .SecretsBase64Map "jwt-key.pem" | b64dec | nindent 6 }}
  {{- end }}

  {{- if ne (index .SecretsBase64Map "cert.pem") "" }}
  # Internal "central.stackrox" service TLS certificate.
  serviceTLS:
    cert: |
      {{- index .SecretsBase64Map "cert.pem" | b64dec | nindent 6 }}
    key: |
      {{- index .SecretsBase64Map "key.pem" | b64dec | nindent 6 }}
  {{- end }}

  {{- if ne (index .SecretsBase64Map "default-tls.crt") "" }}
  # Default, i.e., user-visible certificate.
  defaultTLS:
    cert: |
      {{- index .SecretsBase64Map "default-tls.crt" | b64dec | nindent 6 }}
    key: |
      {{- index .SecretsBase64Map "default-tls.key" | b64dec | nindent 6 }}
  {{- end }}

scanner:
  {{- if ne (index .SecretsBase64Map "scanner-db-password") "" }}
  # Password for securing the communication between Scanner and its DB.
  # This password is not relevant to the user (unless for debugging purposes);
  # it merely acts as a pre-shared, random secret for securing the connection.
  dbPassword:
    value: {{ index .SecretsBase64Map "scanner-db-password" | b64dec }}
  {{- end }}
 
  {{- if ne (index .SecretsBase64Map "scanner-cert.pem") "" }}
  # Internal "scanner.stackrox.svc" service TLS certificate.
  serviceTLS:
    cert: |
      {{- index .SecretsBase64Map "scanner-cert.pem" | b64dec | nindent 6 }}
    key: |
      {{- index .SecretsBase64Map "scanner-key.pem" | b64dec | nindent 6 }}
  {{- end }}

  {{- if ne (index .SecretsBase64Map "scanner-db-cert.pem") "" }}
  # Internal "scanner-db.stackrox" service TLS certificate.
  dbServiceTLS:
    cert: |
      {{- index .SecretsBase64Map "scanner-db-cert.pem" | b64dec | nindent 6 }}
    key: |
      {{- index .SecretsBase64Map "scanner-db-key.pem" | b64dec | nindent 6 }}
  {{- end }}
`

	privateValuesYamlPostgresTemplateStr = `
# StackRox Central Services chart - SECRET configuration values.
#
# These are secret values for the deployment of the StackRox Central Services chart.
# Store this file in a safe place, such as a secrets management system.
# Note that these values are usually NOT required when upgrading or applying configuration
# changes, but they are required for re-deploying an exact copy to a separate cluster.

{{- if ne (index .SecretsBase64Map "ca.pem") "" }}
# Internal service TLS Certificate Authority
ca:
  cert: |
    {{- index .SecretsBase64Map "ca.pem" | b64dec | nindent 4 }}
  key: |
    {{- index .SecretsBase64Map "ca-key.pem" | b64dec | nindent 4 }}
{{- end }}

{{- if ne (index .SecretsBase64Map "central-license") "" }}
# StackRox license key
licenseKey: | 
  {{- index .SecretsBase64Map "central-license" | b64dec | nindent 2 }}
{{- end }}

# Configuration secrets for the Central deployment
central:
  {{- if ne (index .SecretsBase64Map "htpasswd") "" }}
  # Administrator password for logging in to the StackRox Portal.
  # htpasswd (bcrypt) encoded for security reasons, consult the "password" file
  # that is part of the deployment bundle for the raw password.
  adminPassword:
    htpasswd: |
      {{- index .SecretsBase64Map "htpasswd" | b64dec | nindent 6 }}
  {{- end }}

  {{- if ne (index .SecretsBase64Map "central-db-password") "" }}
  # Password for securing the communication between Central and its DB.
  # This password is not relevant to the user (unless for debugging purposes);
  # it merely acts as a pre-shared, random secret for securing the connection.
  dbPassword:
    value: {{ index .SecretsBase64Map "central-db-password" | b64dec }}
  {{- end }}

  {{- if ne (index .SecretsBase64Map "jwt-key.pem") "" }}
  # Private key used for signing JWT tokens.
  jwtSigner:
    key: |
      {{- index .SecretsBase64Map "jwt-key.pem" | b64dec | nindent 6 }}
  {{- end }}

  {{- if ne (index .SecretsBase64Map "cert.pem") "" }}
  # Internal "central.stackrox" service TLS certificate.
  serviceTLS:
    cert: |
      {{- index .SecretsBase64Map "cert.pem" | b64dec | nindent 6 }}
    key: |
      {{- index .SecretsBase64Map "key.pem" | b64dec | nindent 6 }}
  {{- end }}

  {{- if ne (index .SecretsBase64Map "default-tls.crt") "" }}
  # Default, i.e., user-visible certificate.
  defaultTLS:
    cert: |
      {{- index .SecretsBase64Map "default-tls.crt" | b64dec | nindent 6 }}
    key: |
      {{- index .SecretsBase64Map "default-tls.key" | b64dec | nindent 6 }}
  {{- end }}

scanner:
  {{- if ne (index .SecretsBase64Map "scanner-db-password") "" }}
  # Password for securing the communication between Scanner and its DB.
  # This password is not relevant to the user (unless for debugging purposes);
  # it merely acts as a pre-shared, random secret for securing the connection.
  dbPassword:
    value: {{ index .SecretsBase64Map "scanner-db-password" | b64dec }}
  {{- end }}
 
  {{- if ne (index .SecretsBase64Map "scanner-cert.pem") "" }}
  # Internal "scanner.stackrox.svc" service TLS certificate.
  serviceTLS:
    cert: |
      {{- index .SecretsBase64Map "scanner-cert.pem" | b64dec | nindent 6 }}
    key: |
      {{- index .SecretsBase64Map "scanner-key.pem" | b64dec | nindent 6 }}
  {{- end }}

  {{- if ne (index .SecretsBase64Map "scanner-db-cert.pem") "" }}
  # Internal "scanner-db.stackrox" service TLS certificate.
  dbServiceTLS:
    cert: |
      {{- index .SecretsBase64Map "scanner-db-cert.pem" | b64dec | nindent 6 }}
    key: |
      {{- index .SecretsBase64Map "scanner-db-key.pem" | b64dec | nindent 6 }}
  {{- end }}
`
)

var (
	publicValuesTemplate = template.Must(
		helmTemplate.InitTemplate("values-public.yaml").Parse(publicValuesTemplateStr))

	privateValuesTemplate = template.Must(
		helmTemplate.InitTemplate("values-private.yaml").Parse(privateValuesYamlTemplateStr))

	privateValuesPostgresTemplate = template.Must(
		helmTemplate.InitTemplate("values-private.yaml").Parse(privateValuesYamlPostgresTemplateStr))
)

// renderNewHelmValues creates values files for the new Central Services helm charts,
// based on the given config. The values are returned as a *zip.File slice, containing
// two entries, one for `values-public.yaml`, and one for `values-private.yaml`.
func renderNewHelmValues(c Config) ([]*zip.File, error) {
	privateTemplate := privateValuesTemplate
	if features.PostgresDatastore.Enabled() {
		privateTemplate = privateValuesPostgresTemplate
	}

	publicValuesBytes, err := templates.ExecuteToBytes(publicValuesTemplate, &c)
	if err != nil {
		return nil, errors.Wrap(err, "executing public values template")
	}
	privateValuesBytes, err := templates.ExecuteToBytes(privateTemplate, &c)
	if err != nil {
		return nil, errors.Wrap(err, "executing private values template")
	}

	files := []*zip.File{
		zip.NewFile("values-public.yaml", publicValuesBytes, 0),
		zip.NewFile("values-private.yaml", privateValuesBytes, zip.Sensitive),
	}
	return files, nil
}

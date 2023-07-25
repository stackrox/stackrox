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
  telemetry:
    enabled: {{ .K8sConfig.Telemetry.Enabled }}
    storage:
      endpoint: {{ .K8sConfig.Telemetry.StorageEndpoint }}
      key: {{ .K8sConfig.Telemetry.StorageKey }}
  {{- if or .K8sConfig.DeclarativeConfigMounts.ConfigMaps .K8sConfig.DeclarativeConfigMounts.Secrets }}
  declarativeConfiguration:
    mounts:
      {{- if .K8sConfig.DeclarativeConfigMounts.ConfigMaps }}
      configMaps:
      {{- range .K8sConfig.DeclarativeConfigMounts.ConfigMaps }}
      - {{ . | quote }}
      {{- end }}
      {{- end }}
      {{- if .K8sConfig.DeclarativeConfigMounts.Secrets }}
      secrets:
      {{- range .K8sConfig.DeclarativeConfigMounts.Secrets }}
      - {{ . | quote }}
      {{- end }}
      {{- end }}
  {{- end }}

  {{- if ne (.GetConfigOverride "endpoints.yaml") "" }}
  endpointsConfig: |
    {{- .GetConfigOverride "endpoints.yaml" | nindent 4 }}
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

  persistence:
    none: true

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

  db:
    enabled: true
    {{- if .HasCentralDBHostPath }}
    {{- if .HostPath.DB.WithNodeSelector }}
    nodeSelector:
      {{ .HostPath.DB.NodeSelectorKey | quote }}: {{ .HostPath.DB.NodeSelectorValue | quote }}
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
      {{- if .HasCentralDBHostPath }}
      hostPath: {{ .HostPath.DB.HostPath }}
      {{ else if .HasCentralDBExternal }}
      persistentVolumeClaim:
        claimName: {{ .External.DB.Name | quote }}
        size: {{ printf "%dGi" .External.DB.Size | quote }}
        {{- if .External.DB.StorageClass }}
        storageClass: {{ .External.DB.StorageClass | quote }}
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
{{- $_ := unset $envVars "ROX_TELEMETRY_ENDPOINT" -}}
{{- $_ := unset $envVars "ROX_TELEMETRY_STORAGE_KEY_V1" -}}
{{- if $envVars }}

customize:
  # Custom environment variables that will be applied to all containers
  # of all workloads.
  envVars:
    {{ range $key, $value := $envVars -}}
    {{ quote $key }}: {{ quote $value }}
    {{ end }}

{{- end }}

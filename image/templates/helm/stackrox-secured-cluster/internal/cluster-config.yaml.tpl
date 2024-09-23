{{- if ._rox.clusterName }}
clusterName: {{ ._rox.clusterName }}
{{- end }}
managedBy: {{ ._rox.managedBy }}
notHelmManaged: {{ eq ._rox.managedBy "MANAGER_TYPE_MANUAL" }}
clusterConfig:
  staticConfig:
    {{- if not ._rox.env.openshift }}
    type: KUBERNETES_CLUSTER
    {{- else }}
    type: {{ if eq (int ._rox.env.openshift) 4 -}} OPENSHIFT4_CLUSTER {{- else -}} OPENSHIFT_CLUSTER {{ end }}
    {{- end }}
    mainImage: {{ coalesce ._rox.image.main._abbrevImageRef ._rox.image.main.fullRef }}
    collectorImage: {{ coalesce ._rox.image.collector._abbrevImageRef ._rox.image.collector.fullRef }}
    centralApiEndpoint: {{ ._rox.centralEndpoint }}
    collectionMethod: {{ ._rox.collector.collectionMethod | upper | replace "-" "_" }}
    admissionController: {{ ._rox.admissionControl.listenOnCreates }}
    admissionControllerUpdates: {{ ._rox.admissionControl.listenOnUpdates }}
    admissionControllerEvents: {{ ._rox.admissionControl.listenOnEvents }}
    tolerationsConfig:
      disabled: {{ ._rox.collector.disableTaintTolerations }}
    slimCollector: {{ ._rox.collector.slimMode }}
  dynamicConfig:
    disableAuditLogs: {{ ._rox.auditLogs.disableCollection | not | not }}
    admissionControllerConfig:
      enabled: {{ ._rox.admissionControl.dynamic.enforceOnCreates }}
      timeoutSeconds: {{ ._rox.admissionControl.dynamic.timeout }}
      scanInline: {{ ._rox.admissionControl.dynamic.scanInline }}
      disableBypass: {{ ._rox.admissionControl.dynamic.disableBypass }}
      enforceOnUpdates: {{ ._rox.admissionControl.dynamic.enforceOnUpdates }}
    registryOverride: {{ ._rox.registryOverride }}
    collectorConfig:
      process_config:
        enabled: {{ ._rox.processConfig.enabled | default true }}
      network_connection_config:
        enabled: {{ ._rox.networkEndpointConfig.enabled | default true }}
        aggregate_external: {{ .Values.networkConnectionConfig.aggregateExternal | default true }}
      network_endpoint_config:
        enabled: {{ ._rox.networkEndpointConfig.enabled | default true }}
        include_listening_endpoint_processes: {{ ._rox.networkEndpointConfig.includeListeningEndpointProcesses | default false }}
      namespace_scope_config:
        {{- range ._rox.namespaceScopeConfig }}
        feature:
          network_connection:
            enabled: {{ .feature.networkConnection.enabled | default true }}
            aggregate_external: {{ .feature.networkConnection.aggregateExternal | default true }}
        namespace_selection:
          {{- range .namespaceSelection }}
          - namespace: {{ .namespace }}
            match_type: {{ .matchType }}
          {{- end }}
        {{- end }}
  configFingerprint: {{ ._rox._configFP }}
  clusterLabels: {{- toYaml ._rox.clusterLabels | nindent 4 }}

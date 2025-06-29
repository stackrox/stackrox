networking:
  externalIps:
    {{- $enabled := lower (default "not_set" ._rox.collector.runtimeConfig.networking.externalIps.enabled) }}
    {{- if or (eq $enabled "enabled") (eq $enabled "true") }}
    enabled: ENABLED
    {{- else if or (eq $enabled "disabled") (eq $enabled "false") }}
    enabled: DISABLED
    {{- end }}

  maxConnectionsPerMinute: {{ ._rox.collector.runtimeConfig.networking.maxConnectionsPerMinute }}

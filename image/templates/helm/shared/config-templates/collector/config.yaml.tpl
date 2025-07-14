networking:
  externalIps:
    {{- $enabled := ._rox.collector.runtimeConfig.networking.externalIps.enabled }}
    {{- if not (kindIs "invalid" $enabled) }}
      {{- $enabledStr := lower (print $enabled) }}
      {{- if or (eq $enabledStr "enabled") (eq $enabledStr "true") }}
    enabled: ENABLED
      {{- else if or (eq $enabledStr "disabled") (eq $enabledStr "false") }}
    enabled: DISABLED
      {{- end }}
    {{- end }}

  maxConnectionsPerMinute: {{ ._rox.collector.runtimeConfig.networking.maxConnectionsPerMinute }}

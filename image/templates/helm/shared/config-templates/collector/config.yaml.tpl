networking:
  externalIps:
    {{- $enabled := ._rox.collector.runtimeConfig.networking.externalIps.enabled }}
    {{- if not (kindIs "invalid" $enabled) }}
      {{- $enabledStr := lower (print $enabled) }}
      {{- if eq $enabledStr "enabled" }}
    enabled: ENABLED
      {{- else if eq $enabledStr "disabled" }}
    enabled: DISABLED
      {{- end }}
    {{- end }}

  maxConnectionsPerMinute: {{ ._rox.collector.runtimeConfig.networking.maxConnectionsPerMinute }}

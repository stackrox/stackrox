{{- /*
  This is the configuration file template for Collector.
  Except for in extremely rare circumstances, you DO NOT need to modify this file.
  All config options that are possibly dynamic are templated out and can be modified
  via `--set`/values-files specified via `-f`.
     */ -}}

# Configuration file for collector.

networking:
  externalIps:
    {{- $enabled := lower (default "disabled" ._rox.collector.runtimeConfig.networking.externalIps.enabled) }}
    {{- if eq $enabled "auto" }}
    enabled: AUTO
    {{- else if eq $enabled "disabled" }}
    enabled: DISABLED
    {{- else if eq $enabled "enabled" }}
    enabled: ENABLED
    {{- else if eq $enabled "true" }}
    enabled: ENABLED
    {{- else if eq $enabled "false" }}
    enabled: DISABLED
    {{- else }}
    enabled: DISABLED
    {{- end }}

  maxConnectionsPerMinute: {{ ._rox.collector.runtimeConfig.networking.maxConnectionsPerMinute }}

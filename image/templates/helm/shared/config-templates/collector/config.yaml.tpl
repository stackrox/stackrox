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
    enabled: Auto
    {{- else if eq $enabled "disabled" }}
    enabled: Disabled
    {{- else if eq $enabled "enabled" }}
    enabled: Enabled
    {{- else if eq $enabled "true" }}
    enabled: Enabled
    {{- else if eq $enabled "false" }}
    enabled: Disabled
    {{- else }}
    enabled: Disabled
    {{- end }}

  maxConnectionsPerMinute: {{ ._rox.collector.runtimeConfig.networking.maxConnectionsPerMinute }}

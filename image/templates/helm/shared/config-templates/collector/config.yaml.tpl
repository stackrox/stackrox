{{- /*
  This is the configuration file template for Collecto.
  Except for in extremely rare circumstances, you DO NOT need to modify this file.
  All config options that are possibly dynamic are templated out and can be modified
  via `--set`/values-files specified via `-f`.
     */ -}}

{{- include "srox.init" . -}}
# Configuration file for collector.

networking:
  externalIps:
    enable: {{ ._rox.collector.runtimeConfig.networking.externalIps.enable | default false }}
  per_container_rate_limit: {{ ._rox.collector.runtimeConfig.networking.perContainerRateLimit | default 1024 }}

{{- /*
  This is the configuration file template for Collector.
  Except for in extremely rare circumstances, you DO NOT need to modify this file.
  All config options that are possibly dynamic are templated out and can be modified
  via `--set`/values-files specified via `-f`.
     */ -}}

# Configuration file for collector.

networking:
  externalIps:
    enable: {{ ._rox.collector.runtimeConfig.networking.externalIps.enabled }}
  perContainerRateLimit: {{ ._rox.collector.runtimeConfig.networking.perContainerRateLimit }}

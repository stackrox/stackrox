{{- /*
  This is the configuration file template for Collecto.
  Except for in extremely rare circumstances, you DO NOT need to modify this file.
  All config options that are possibly dynamic are templated out and can be modified
  via `--set`/values-files specified via `-f`.
     */ -}}

# Configuration file for collector.

networking:
  externalIps:
    enable: false
  perContainerRateLimit: 1024

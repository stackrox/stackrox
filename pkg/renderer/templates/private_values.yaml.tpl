# StackRox Central Services chart - SECRET configuration values.
#
# These are secret values for the deployment of the StackRox Central Services chart.
# Store this file in a safe place, such as a secrets management system.
# Note that these values are usually NOT required when upgrading or applying configuration
# changes, but they are required for re-deploying an exact copy to a separate cluster.

{{- if ne (index .SecretsBase64Map "ca.pem") "" }}
# Internal service TLS Certificate Authority
ca:
  cert: |
    {{- index .SecretsBase64Map "ca.pem" | b64dec | nindent 4 }}
  key: |
    {{- index .SecretsBase64Map "ca-key.pem" | b64dec | nindent 4 }}
{{- end }}

{{- if ne (index .SecretsBase64Map "central-license") "" }}
# StackRox license key
licenseKey: |
  {{- index .SecretsBase64Map "central-license" | b64dec | nindent 2 }}
{{- end }}

# Configuration secrets for the Central deployment
central:
  {{- if ne (index .SecretsBase64Map "htpasswd") "" }}
  # Administrator password for logging in to the StackRox Portal.
  # htpasswd (bcrypt) encoded for security reasons, consult the "password" file
  # that is part of the deployment bundle for the raw password.
  adminPassword:
    htpasswd: |
      {{- index .SecretsBase64Map "htpasswd" | b64dec | nindent 6 }}
  {{- end }}

  {{- if ne (index .SecretsBase64Map "jwt-key.pem") "" }}
  # Private key used for signing JWT tokens.
  jwtSigner:
    key: |
      {{- index .SecretsBase64Map "jwt-key.pem" | b64dec | nindent 6 }}
  {{- end }}

  {{- if ne (index .SecretsBase64Map "cert.pem") "" }}
  # Internal "central.stackrox" service TLS certificate.
  serviceTLS:
    cert: |
      {{- index .SecretsBase64Map "cert.pem" | b64dec | nindent 6 }}
    key: |
      {{- index .SecretsBase64Map "key.pem" | b64dec | nindent 6 }}
  {{- end }}

  {{- if ne (index .SecretsBase64Map "default-tls.crt") "" }}
  # Default, i.e., user-visible certificate.
  defaultTLS:
    cert: |
      {{- index .SecretsBase64Map "default-tls.crt" | b64dec | nindent 6 }}
    key: |
      {{- index .SecretsBase64Map "default-tls.key" | b64dec | nindent 6 }}
  {{- end }}

scanner:
  {{- if ne (index .SecretsBase64Map "scanner-db-password") "" }}
  # Password for securing the communication between Scanner and its DB.
  # This password is not relevant to the user (unless for debugging purposes);
  # it merely acts as a pre-shared, random secret for securing the connection.
  dbPassword:
    value: {{ index .SecretsBase64Map "scanner-db-password" | b64dec }}
  {{- end }}

  {{- if ne (index .SecretsBase64Map "scanner-cert.pem") "" }}
  # Internal "scanner.stackrox.svc" service TLS certificate.
  serviceTLS:
    cert: |
      {{- index .SecretsBase64Map "scanner-cert.pem" | b64dec | nindent 6 }}
    key: |
      {{- index .SecretsBase64Map "scanner-key.pem" | b64dec | nindent 6 }}
  {{- end }}

  {{- if ne (index .SecretsBase64Map "scanner-db-cert.pem") "" }}
  # Internal "scanner-db.stackrox" service TLS certificate.
  dbServiceTLS:
    cert: |
      {{- index .SecretsBase64Map "scanner-db-cert.pem" | b64dec | nindent 6 }}
    key: |
      {{- index .SecretsBase64Map "scanner-db-key.pem" | b64dec | nindent 6 }}
  {{- end }}

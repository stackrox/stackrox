#! /bin/sh

set -ex

# overwriting the config makes this script idempotent to restarts
cp /etc/prometheus/prometheus.yml.template /etc/prometheus/prometheus.yml

if [[ -n "${ROX_CENTRAL_ENDPOINT}" ]]; then
cat <<EOF >> /etc/prometheus/prometheus.yml
- job_name: central
  scrape_interval: 15s
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: https
  static_configs:
  - targets:
    - ${ROX_CENTRAL_ENDPOINT}
  tls_config:
    insecure_skip_verify: true
EOF
fi

if [[ -n "${ROX_SENSOR_ENDPOINT}" ]]; then
cat <<EOF >> /etc/prometheus/prometheus.yml
- job_name: sensor
  scrape_interval: 15s
  scrape_timeout: 10s
  metrics_path: /metrics
  scheme: https
  static_configs:
  - targets:
    - ${ROX_SENSOR_ENDPOINT}
  tls_config:
    insecure_skip_verify: true
EOF
fi

exec prometheus --config.file=/etc/prometheus/prometheus.yml

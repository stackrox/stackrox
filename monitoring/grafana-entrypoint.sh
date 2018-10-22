#! /bin/sh

PASSWORD=$(cat /run/secrets/stackrox.io/monitoring/password)

cat <<EOF > /etc/grafana/provisioning/datasources/influxdb.yaml
apiVersion: 1

datasources:
- name: InfluxDB
  type: influxdb
  access: proxy
  orgId: 1
  url: https://localhost:8086
  password: ${PASSWORD}
  user: telegraf
  database: telegraf
  isDefault: true
  jsonData:
    tlsSkipVerify: true
  version: 1
  editable: false
EOF

exec $@
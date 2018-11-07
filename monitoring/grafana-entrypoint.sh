#! /bin/sh

PASSWORD=$(cat /run/secrets/stackrox.io/monitoring/password)

cat <<EOF > /etc/grafana/provisioning/datasources/influxdb.yaml
apiVersion: 1

datasources:
- name: InfluxDB 12h
  type: influxdb
  access: proxy
  orgId: 1
  url: https://localhost:8086
  password: ${PASSWORD}
  user: telegraf
  database: telegraf_12h
  isDefault: true
  jsonData:
    tlsSkipVerify: true
  version: 1
  editable: false
- name: InfluxDB 2w
  type: influxdb
  access: proxy
  orgId: 1
  url: https://localhost:8086
  password: ${PASSWORD}
  user: telegraf
  database: telegraf_2w
  isDefault: false
  jsonData:
    tlsSkipVerify: true
  version: 1
  editable: false
- name: InfluxDB forever
  type: influxdb
  access: proxy
  orgId: 1
  url: https://localhost:8086
  password: ${PASSWORD}
  user: telegraf
  database: telegraf_forever
  isDefault: false
  jsonData:
    tlsSkipVerify: true
  version: 1
  editable: false
EOF

exec $@
#! /bin/sh

INFLUXDB_ADMIN_USER=telegraf
INFLUXDB_ADMIN_PASSWORD=$(cat /run/secrets/stackrox.io/monitoring/client/password)

QUERY="CREATE USER \"$INFLUXDB_ADMIN_USER\" WITH PASSWORD '$INFLUXDB_ADMIN_PASSWORD' WITH ALL PRIVILEGES"
sleep 10

/influx -ssl -unsafeSsl -execute "$QUERY"

#! /bin/sh

cp /run/secrets/stackrox.io/monitoring/ca.pem /usr/local/share/ca-certificates/ca.crt
update-ca-certificates

export PASSWORD=$(cat /run/secrets/stackrox.io/monitoring/password)
exec /telegraf

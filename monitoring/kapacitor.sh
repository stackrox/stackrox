#! /bin/sh

set -ex

PASSWORD=$(cat /run/secrets/stackrox.io/monitoring/password)

cp /etc/kapacitor/kapacitor.conf kapacitor.conf.tmp

echo "s/PASSWORD/$PASSWORD/g" > sed.txt
sed -i -f sed.txt kapacitor.conf.tmp

exec /kapacitord -config kapacitor.conf.tmp

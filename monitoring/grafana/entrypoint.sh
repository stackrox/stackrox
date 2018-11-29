#! /bin/sh

PASSWORD=$(cat /run/secrets/stackrox.io/monitoring/secrets/password)

cat <<EOF > /etc/grafana/grafana.ini
[server]
# Protocol (http, https, socket)
protocol = https

# The ip address to bind to, empty will bind to all interfaces
;http_addr =

# The http port  to use
http_port = 8443

# https certs & key file
cert_file = "/run/secrets/stackrox.io/monitoring/certs/cert.pem"
cert_key = "/run/secrets/stackrox.io/monitoring/certs/key.pem"

[security]
# default admin user, created on startup
;admin_user = admin

# default admin password, can be changed before first start of grafana,  or in profile settings
admin_password = $PASSWORD

EOF

exec $@

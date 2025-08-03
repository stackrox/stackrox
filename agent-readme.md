# agent

## set up agent in vm

SCP certs into VM

podman run --tls-verify=false -v ./certs:/certs -e=ROX_MTLS_KEY_FILE=/certs/key.pem -e=ROX_MTLS_CERT_FILE=/certs/cert.pem --env=ROX_MTLS_CA_FILE=/certs/ca.pem -e=ROX_SENSOR_ENDPOINT=sensor.default.svc:443 kind-registry:5000/stackrox/stackrox-agent:latest@sha256:04d97d53abf07d69ccaeae838472f1b67700584659ee77a8493b96eede46a168

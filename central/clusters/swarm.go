package clusters

import (
	"strconv"
	"text/template"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

func init() {
	deployers[v1.ClusterType_SWARM_CLUSTER] = newSwarm()
	deployers[v1.ClusterType_DOCKER_EE_CLUSTER] = newSwarm()
}

func newSwarm() deployer {
	return &basicDeployer{
		deploy:    template.Must(template.New("swarm").Parse(swarmDeploy)),
		cmd:       template.Must(template.New("swarm").Parse(swarmCmd)),
		addFields: addSwarmFields,
	}
}

func addSwarmFields(c Wrap, fields map[string]string) {
	fields["DisableSwarmTLS"] = strconv.FormatBool(c.DisableSwarmTls)
}

var (
	swarmDeploy = `version: "3.2"
services:
  sensor:
    image: {{.Image}}
    entrypoint:
      - swarm-sensor
    networks:
      net:
    deploy:
      labels:
        owner: stackrox
        email: support@stackrox.com
      placement:
        constraints:
          - node.role==manager
    volumes:
      - type: bind
        source: /var/run/docker.sock
        target: /var/run/docker.sock
    environment:
      - "{{.PublicEndpointEnv}}={{.PublicEndpoint}}"
      - "{{.ClusterIDEnv}}={{.ClusterID}}"
      - "{{.AdvertisedEndpointEnv}}={{.AdvertisedEndpoint}}"
      - "{{.ImageEnv}}={{.Image}}"
{{ if ne .DisableSwarmTLS "true" }}
      - "DOCKER_CERT_PATH=/run/secrets/stackrox.io/docker/"
      - "DOCKER_HOST=$DOCKER_HOST"
      - "DOCKER_TLS_VERIFY=$DOCKER_TLS_VERIFY"
{{ end }}
    secrets:
      - source: sensor_certificate
        target: stackrox.io/cert.pem
        mode: 400
      - source: sensor_private_key
        target: stackrox.io/key.pem
        mode: 400
      - source: central_certificate
        target: stackrox.io/ca.pem
        mode: 400
{{ if ne .DisableSwarmTLS "true" }}
      - source: docker_client_ca_pem
        target: stackrox.io/docker/ca.pem
        mode: 400
      - source: docker_client_cert_pem
        target: stackrox.io/docker/cert.pem
        mode: 400
      - source: docker_client_key_pem
        target: stackrox.io/docker/key.pem
        mode: 400
{{ end }}
      - source: registry_auth
        target: stackrox.io/registry_auth
        mode: 400
networks:
  net:
    driver: overlay
    attachable: true
secrets:
  sensor_private_key:
    file: sensor-key.pem
  sensor_certificate:
    file: sensor-cert.pem
  central_certificate:
    file: central-ca.pem
{{ if ne .DisableSwarmTLS "true" }}
  docker_client_ca_pem:
    file: docker-ca.pem
  docker_client_cert_pem:
    file: docker-cert.pem
  docker_client_key_pem:
    file: docker-key.pem
{{ end }}
  registry_auth:
    file: registry-auth
`

	swarmCmd = commandPrefix + `WD=$(pwd)
cd $DIR

# Create registry-auth secret, used to pull the benchmark image.
if [ -z "$REGISTRY_USERNAME" ]; then
  echo -n "Registry username for StackRox Prevent image: "
  read -s REGISTRY_USERNAME
  echo
fi
if [ -z "$REGISTRY_PASSWORD" ]; then
  echo -n "Registry password for StackRox Prevent image: "
  read -s REGISTRY_PASSWORD
  echo
fi
REGISTRY_AUTH="{\"username\": \"$REGISTRY_USERNAME\", \"password\": \"$REGISTRY_PASSWORD\"}"
echo -n "$REGISTRY_AUTH" | base64 | tr -- '+=/' '-_~' > registry-auth

# Gather client cert bundle if it is present.
if [ -n "$DOCKER_CERT_PATH" ]; then
  cp $DOCKER_CERT_PATH/ca.pem ./docker-ca.pem
  cp $DOCKER_CERT_PATH/key.pem ./docker-key.pem
  cp $DOCKER_CERT_PATH/cert.pem ./docker-cert.pem
fi

# Deploy.
docker stack deploy -c ./sensor-deploy.yaml prevent --with-registry-auth

# Clean up temporary files.
rm registry-auth
if [ -n "$DOCKER_CERT_PATH" ]; then
  rm ./docker-*
fi

cd $WD
`
)

package clusters

import (
	"text/template"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

func init() {
	deployers[v1.ClusterType_SWARM_CLUSTER] = newSwarm()
	deployers[v1.ClusterType_DOCKER_EE_CLUSTER] = newSwarm()
}

func newSwarm() deployer {
	return &basicDeployer{
		deploy: template.Must(template.New("swarm").Parse(swarmDeploy)),
		cmd:    template.Must(template.New("swarm").Parse(swarmCmd)),
	}
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
      placement:
        constraints:
          - node.role==manager
    volumes:
      - type: bind
        source: /var/run/docker.sock
        target: /var/run/docker.sock
    environment:
      - "{{.PublicEndpointEnv}}={{.PublicEndpoint}}"
      - "{{.ClusterNameEnv}}={{.ClusterName}}"
      - "{{.AdvertisedEndpointEnv}}={{.AdvertisedEndpoint}}"
      - "{{.ImageEnv}}={{.Image}}"
      - "DOCKER_CERT_PATH=/run/secrets/stackrox.io/docker/"
      - "DOCKER_HOST=$DOCKER_HOST"
      - "DOCKER_TLS_VERIFY=$DOCKER_TLS_VERIFY"
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
      - source: docker_client_ca_pem
        target: stackrox.io/docker/ca.pem
        mode: 400
      - source: docker_client_cert_pem
        target: stackrox.io/docker/cert.pem
        mode: 400
      - source: docker_client_key_pem
        target: stackrox.io/docker/key.pem
        mode: 400
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
  docker_client_ca_pem:
    file: docker-ca.pem
  docker_client_cert_pem:
    file: docker-cert.pem
  docker_client_key_pem:
    file: docker-key.pem
  registry_auth:
    file: registry-auth
`

	swarmCmd = commandPrefix + `set -u
WD=$(pwd)
cd $DIR

echo -n "Registry username for StackRox Mitigate image: "
read -s REGISTRY_USERNAME
echo
echo -n "Registry password for StackRox Mitigate image: "
read -s REGISTRY_PASSWORD
echo

REGISTRY_AUTH="{\"username\": \"$REGISTRY_USERNAME\", \"password\": \"$REGISTRY_PASSWORD\"}"
echo -n "$REGISTRY_AUTH" | base64 | tr -- '+=/' '-_~' > registry-auth

cp $DOCKER_CERT_PATH/ca.pem ./docker-ca.pem
cp $DOCKER_CERT_PATH/key.pem ./docker-key.pem
cp $DOCKER_CERT_PATH/cert.pem ./docker-cert.pem

docker stack deploy -c ./sensor-deploy.yaml mitigate --with-registry-auth

rm ./docker-*
rm registry-auth

cd $WD
`
)

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
    file: central-ca.pem`
	// TODO(cg): Do we need to include DOCKER_HOST, DOCKER_CERT_PATH, DOCKER_TLS_VERIFY?

	swarmCmd = commandPrefix + `WD=$(pwd)
cd $DIR
docker stack deploy -c ./sensor-deploy.yaml apollo
cd $WD
`
)

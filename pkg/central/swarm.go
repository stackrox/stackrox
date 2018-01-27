package central

import (
	"text/template"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

func init() {
	Deployers[v1.ClusterType_SWARM_CLUSTER] = newSwarm()
	Deployers[v1.ClusterType_DOCKER_EE_CLUSTER] = newSwarm()
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
  central:
    image: {{.Image}}
    entrypoint: ["central"]
    networks:
      net:
    ports:
      - target: 443
        published: {{.PublicPort}}
        protocol: tcp
        mode: ingress
    secrets:
      - source: mitigate_private_key
        target: stackrox.io/ca-key.pem
        mode: 400
      - source: mitigate_certificate
        target: stackrox.io/ca.pem
        mode: 400
networks:
  net:
    driver: overlay
    attachable: true
secrets:
  mitigate_private_key:
    file: ./ca-key.pem
  mitigate_certificate:
    file: ./ca.pem
`

	swarmCmd = commandPrefix + `WD=$(pwd)
cd $DIR

docker stack deploy -c ./deploy.yaml mitigate --with-registry-auth

cd $WD
`
)

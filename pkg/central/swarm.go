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
    deploy:
       labels: [owner=stackrox,email=support@stackrox.com]
       resources:
         reservations:
           cpus: '1.0'
           memory: 2G
         limits:
           cpus: '2.0'
           memory: 8G
    ports:
      - target: 443
        published: {{.PublicPort}}
        protocol: tcp
        mode: ingress
    secrets:
      - source: prevent_private_key
        target: stackrox.io/ca-key.pem
        mode: 400
      - source: prevent_certificate
        target: stackrox.io/ca.pem
        mode: 400
networks:
  net:
    driver: overlay
    attachable: true
secrets:
  prevent_private_key:
    file: ./ca-key.pem
  prevent_certificate:
    file: ./ca.pem
`

	swarmCmd = commandPrefix + `WD=$(pwd)
cd $DIR

docker stack deploy -c ./deploy.yaml prevent --with-registry-auth

cd $WD
`
)

package clusters

import (
	"strconv"
	"text/template"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/zip"
)

func init() {
	deployers[v1.ClusterType_SWARM_CLUSTER] = newSwarm()
	deployers[v1.ClusterType_DOCKER_EE_CLUSTER] = newSwarm()
}

type swarm struct {
	deploy *template.Template
	cmd    *template.Template
}

func newSwarm() Deployer {
	return &swarm{
		deploy: template.Must(template.New("swarm").Parse(swarmDeploy)),
		cmd:    template.Must(template.New("swarm").Parse(swarmCmd)),
	}
}

func (s *swarm) Render(c Wrap) ([]*v1.File, error) {
	var swarmParams *v1.SwarmParams
	clusterSwarm, ok := c.OrchestratorParams.(*v1.Cluster_Swarm)
	if ok {
		swarmParams = clusterSwarm.Swarm
	}

	fields := fieldsFromWrap(c)
	fields["DisableSwarmTLS"] = strconv.FormatBool(swarmParams.GetDisableSwarmTls())

	var files []*v1.File
	data, err := executeTemplate(s.deploy, fields)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("deploy.yaml", data, false))

	data, err = executeTemplate(s.cmd, fields)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("deploy.sh", data, true))
	files = append(files, zip.NewFile("delete.sh", swarmDelete, true))
	return files, nil
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
{{if eq .DisableSwarmTLS "true" }}
      placement:
        constraints:
          - node.role==manager
{{- end}}
      resources:
        reservations:
          cpus: '0.2'
          memory: 200M
        limits:
          cpus: '0.5'
          memory: 500M
    volumes:
      - type: bind
        source: /var/run/docker.sock
        target: /var/run/docker.sock
    environment:
      - "{{.PublicEndpointEnv}}={{.PublicEndpoint}}"
      - "{{.ClusterIDEnv}}={{.ClusterID}}"
      - "{{.AdvertisedEndpointEnv}}={{.AdvertisedEndpoint}}"
      - "{{.ImageEnv}}={{.Image}}"
{{if ne .DisableSwarmTLS "true" }}
      - "DOCKER_CERT_PATH=/run/secrets/stackrox.io/docker/"
      - "DOCKER_HOST=$DOCKER_HOST"
      - "DOCKER_TLS_VERIFY=$DOCKER_TLS_VERIFY"
{{- end}}
    secrets:
      - source: sensor_certificate
        target: stackrox.io/certs/cert.pem
        mode: 400
      - source: sensor_private_key
        target: stackrox.io/certs/key.pem
        mode: 400
      - source: central_certificate
        target: stackrox.io/certs/ca.pem
        mode: 400
{{if ne .DisableSwarmTLS "true" }}
      - source: docker_client_ca_pem
        target: stackrox.io/docker/ca.pem
        mode: 400
      - source: docker_client_cert_pem
        target: stackrox.io/docker/cert.pem
        mode: 400
      - source: docker_client_key_pem
        target: stackrox.io/docker/key.pem
        mode: 400
{{- end}}
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
{{if ne .DisableSwarmTLS "true"}}
  docker_client_ca_pem:
    file: docker-ca.pem
  docker_client_cert_pem:
    file: docker-cert.pem
  docker_client_key_pem:
    file: docker-key.pem
{{- end}}
  registry_auth:
    file: registry-auth
`

	swarmCmd = commandPrefix + `WD=$(pwd)
cd $DIR

# Create registry-auth secret, used to pull the benchmark image.
if [ -z "$REGISTRY_USERNAME" ]; then
  echo -n "Registry username for StackRox Prevent image: "
  read REGISTRY_USERNAME
  echo
fi
if [ -z "$REGISTRY_PASSWORD" ]; then
  echo -n "Registry password for StackRox Prevent image: "
  read -s REGISTRY_PASSWORD
  echo
fi

# unset the host path so we can get the registry auth locally
OLD_DOCKER_HOST="$DOCKER_HOST"
OLD_DOCKER_CERT_PATH="$DOCKER_CERT_PATH"
OLD_DOCKER_TLS_VERIFY="$DOCKER_TLS_VERIFY"
unset DOCKER_HOST DOCKER_CERT_PATH DOCKER_TLS_VERIFY

docker run --rm --entrypoint=base64 -e REGISTRY_USERNAME="$REGISTRY_USERNAME" -e REGISTRY_PASSWORD="$REGISTRY_PASSWORD" {{.Image}} > registry-auth

export DOCKER_HOST="$OLD_DOCKER_HOST"
export DOCKER_CERT_PATH="$OLD_DOCKER_CERT_PATH"
export DOCKER_TLS_VERIFY="$OLD_DOCKER_TLS_VERIFY"


# Gather client cert bundle if it is present.
if [ -n "$DOCKER_CERT_PATH" ]; then
  cp $DOCKER_CERT_PATH/ca.pem ./docker-ca.pem
  cp $DOCKER_CERT_PATH/key.pem ./docker-key.pem
  cp $DOCKER_CERT_PATH/cert.pem ./docker-cert.pem
fi

# Deploy.
docker stack deploy -c ./deploy.yaml prevent --with-registry-auth

# Clean up temporary files.
rm registry-auth
if [ -n "$DOCKER_CERT_PATH" ]; then
  rm ./docker-*
fi

cd $WD
`

	swarmDelete = commandPrefix + `
docker service rm prevent_sensor
docker secret rm prevent_registry_auth prevent_sensor_certificate prevent_sensor_private_key
`
)

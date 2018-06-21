package central

import (
	"text/template"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/zip"
)

func init() {
	Deployers[v1.ClusterType_SWARM_CLUSTER] = newSwarm()
	Deployers[v1.ClusterType_DOCKER_EE_CLUSTER] = newSwarm()
}

type swarm struct {
	clairifyYaml *template.Template
	cmd          *template.Template
	deploy       *template.Template
}

func newSwarm() deployer {
	return &swarm{
		clairifyYaml: template.Must(template.New("swarm").Parse(swarmClairifyYAML)),
		cmd:          template.Must(template.New("swarm").Parse(swarmCmd)),
		deploy:       template.Must(template.New("swarm").Parse(swarmDeploy)),
	}
}

func (s *swarm) Render(c Config) ([]*v1.File, error) {
	var files []*v1.File
	data, err := executeTemplate(s.deploy, c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("deploy.yaml", data, false))

	data, err = executeTemplate(s.cmd, c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("deploy.sh", data, true))

	data, err = executeTemplate(s.clairifyYaml, c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("clairify.yaml", data, false))

	files = append(files, zip.NewFile("clairify.sh", swarmClairifyScript, true))
	return files, nil
}

var (
	swarmDeploy = `version: "3.2"
services:
  central:
    image: {{.SwarmConfig.PreventImage}}
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
       {{if .HostPath -}}
       placement:
         constraints: [{{.HostPath.NodeSelectorKey}} == {{.HostPath.NodeSelectorValue}}]
       {{- end}}
    ports:
      - target: 443
        published: {{.SwarmConfig.PublicPort}}
        protocol: tcp
        mode: ingress
    secrets:
      - source: prevent_private_key
        target: stackrox.io/ca-key.pem
        mode: 400
      - source: prevent_certificate
        target: stackrox.io/ca.pem
        mode: 400
    {{if .HostPath -}}
    volumes:
      - {{.HostPath.HostPath}}:{{.HostPath.MountPath}}
    {{- end}}
    {{if .External -}}
    volumes:
      - {{.External.Name}}:{{.External.MountPath}}
    {{- end}}
networks:
  net:
    driver: overlay
    attachable: true
secrets:
  prevent_private_key:
    file: ./ca-key.pem
  prevent_certificate:
    file: ./ca.pem
{{if .External -}}
volumes:
  {{.External.Name}}:
    external: true
{{- end}}
`

	swarmCmd = commandPrefix + `WD=$(pwd)
cd $DIR

docker stack deploy -c ./deploy.yaml prevent --with-registry-auth

cd $WD
`

	swarmClairifyYAML = `
version: "3.2"
services:
  clairify:
    image: {{.SwarmConfig.ClairifyImage}}
    entrypoint:
      - /init
      - /clairify
    environment:
      - CLAIR_ARGS=-insecure-tls
    networks:
      net:
    deploy:
      labels:
        owner: stackrox
        email: support@stackrox.com
networks:
  net:
    driver: overlay
    attachable: true
`

	swarmClairifyScript = commandPrefix + `
docker stack deploy -c ${DIR}/clairify.yaml prevent --with-registry-auth
`
)

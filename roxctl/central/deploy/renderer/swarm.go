package renderer

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/zip"
)

func init() {
	Deployers[v1.ClusterType_SWARM_CLUSTER] = newSwarm()
	Deployers[v1.ClusterType_DOCKER_EE_CLUSTER] = newSwarm()
}

type swarm struct{}

func newSwarm() deployer {
	return &swarm{}
}

func (s *swarm) Render(c Config) ([]*zip.File, error) {

	filenames := []string{
		"swarm/central.yaml",
		"swarm/central.sh",
		"swarm/clairify.yaml",
		"swarm/clairify.sh",
	}

	var files []*zip.File
	for k, v := range c.SecretsByteMap {
		files = append(files, zip.NewFile(k, v, zip.Sensitive))
	}
	renderedFiles, err := renderFilenames(filenames, &c, "/data/assets/docker-auth.sh")
	if err != nil {
		return nil, err
	}
	return append(files, renderedFiles...), nil
}

func (s *swarm) Instructions(c Config) string {
	return `To deploy:
  1. Unzip the deployment bundle.
  2. Run central.sh.
  3. If you want to run the StackRox Clairify scanner, run clairify.sh.`
}

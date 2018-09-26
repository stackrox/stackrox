package central

import (
	"github.com/stackrox/rox/generated/api/v1"
)

func init() {
	Deployers[v1.ClusterType_SWARM_CLUSTER] = newSwarm()
	Deployers[v1.ClusterType_DOCKER_EE_CLUSTER] = newSwarm()
}

type swarm struct{}

func newSwarm() deployer {
	return &swarm{}
}

func (s *swarm) Render(c Config) ([]*v1.File, error) {

	filenames := []string{
		"swarm/central.yaml",
		"swarm/central.sh",
		"swarm/clairify.yaml",
		"swarm/clairify.sh",
	}

	return renderFilenames(filenames, c)
}

func (s *swarm) Instructions() string {
	return `To deploy:
  1. Unzip the deployment bundle.
  2. Run central.sh.
  3. If you want to run the StackRox Clairify scanner, run clairify.sh.`
}

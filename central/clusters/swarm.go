package clusters

import (
	"strconv"

	"github.com/stackrox/rox/generated/api/v1"
)

func init() {
	deployers[v1.ClusterType_SWARM_CLUSTER] = newSwarm()
	deployers[v1.ClusterType_DOCKER_EE_CLUSTER] = newSwarm()
}

type swarm struct {
}

func newSwarm() Deployer {
	return &swarm{}
}

func (s *swarm) Render(c Wrap) ([]*v1.File, error) {
	var swarmParams *v1.SwarmParams
	clusterSwarm, ok := c.OrchestratorParams.(*v1.Cluster_Swarm)
	if ok {
		swarmParams = clusterSwarm.Swarm
	}

	fields := fieldsFromWrap(c)
	fields["DisableSwarmTLS"] = strconv.FormatBool(swarmParams.GetDisableSwarmTls())

	filenames := []string{
		"swarm/sensor.sh",
		"swarm/sensor.yaml",
		"swarm/delete-sensor.sh",
	}

	return renderFilenames(filenames, fields)
}

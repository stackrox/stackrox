package clusters

import (
	"strconv"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/zip"
)

func init() {
	deployers[storage.ClusterType_SWARM_CLUSTER] = newSwarm()
	deployers[storage.ClusterType_DOCKER_EE_CLUSTER] = newSwarm()
}

type swarm struct {
}

func newSwarm() Deployer {
	return &swarm{}
}

func (s *swarm) Render(c Wrap) ([]*zip.File, error) {
	var swarmParams *storage.SwarmParams
	clusterSwarm, ok := c.OrchestratorParams.(*storage.Cluster_Swarm)
	if ok {
		swarmParams = clusterSwarm.Swarm
	}

	fields, err := fieldsFromWrap(c)
	if err != nil {
		return nil, err
	}
	fields["DisableSwarmTLS"] = strconv.FormatBool(swarmParams.GetDisableSwarmTls())

	filenames := []string{
		"swarm/sensor.sh",
		"swarm/sensor.yaml",
		"swarm/delete-sensor.sh",
	}

	return renderFilenames(filenames, fields, "/data/assets/docker-auth.sh")
}

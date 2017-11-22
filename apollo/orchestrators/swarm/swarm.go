package swarm

import (
	"context"
	"fmt"
	"strings"

	"bitbucket.org/stack-rox/apollo/apollo/orchestrators"
	"bitbucket.org/stack-rox/apollo/apollo/orchestrators/types"
	"bitbucket.org/stack-rox/apollo/pkg/docker"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

var (
	log = logging.New("orchestrators/swarm")
)

type swarmOrchestrator struct {
	dockerClient *client.Client
}

func newSwarm() (types.Orchestrator, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("unable to create docker client: %+v", err)
	}

	if err := docker.NegotiateClientVersionToLatest(client, docker.DefaultAPIVersion); err != nil {
		return nil, err
	}
	return &swarmOrchestrator{
		dockerClient: client,
	}, nil
}

func (s *swarmOrchestrator) Launch(service types.SystemService) (string, error) {
	var mounts []mount.Mount
	for _, m := range service.Mounts {
		spl := strings.Split(m, ":")
		mounts = append(mounts, mount.Mount{
			Type:   "bind",
			Source: spl[0],
			Target: spl[1],
		})
	}

	var global *swarm.GlobalService
	if service.Global {
		global = &swarm.GlobalService{}
	}

	spec := swarm.ServiceSpec{
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: swarm.ContainerSpec{
				Image:  service.Image,
				Env:    service.Envs,
				Mounts: mounts,
			},
			RestartPolicy: &swarm.RestartPolicy{
				Condition: swarm.RestartPolicyConditionNone,
			},
		},
		Mode: swarm.ServiceMode{
			Global: global,
		},
	}
	ctx, cancelFunc := docker.TimeoutContext()
	defer cancelFunc()
	createResp, err := s.dockerClient.ServiceCreate(ctx, spec, dockerTypes.ServiceCreateOptions{})
	if err != nil {
		return "", err
	}
	log.Infof("Swarm Create Resp: %+v", createResp)
	return createResp.ID, nil
}

func (s *swarmOrchestrator) Kill(id string) error {
	return s.dockerClient.ServiceRemove(context.Background(), id)
}

func init() {
	orchestrators.Registry["swarm"] = newSwarm
}

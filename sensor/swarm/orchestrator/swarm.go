package orchestrator

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/stackrox/rox/pkg/benchmarks"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/orchestrators"
)

var (
	log = logging.LoggerForModule()
)

type swarmOrchestrator struct {
	dockerClient *client.Client
}

// New creates a new Swarm orchestrator client.
func New() (orchestrators.Orchestrator, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("unable to create docker client: %+v", err)
	}
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	client.NegotiateAPIVersion(ctx)

	return &swarmOrchestrator{
		dockerClient: client,
	}, nil
}

func (s *swarmOrchestrator) LaunchBenchmark(service orchestrators.SystemService) (string, error) {
	service.Command = []string{benchmarks.BenchmarkBootstrapCommand}
	service.Mounts = []string{"/var/run/docker.sock:/var/run/docker.sock"}
	return s.Launch(service)
}

func (s *swarmOrchestrator) Launch(service orchestrators.SystemService) (string, error) {
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
		Annotations: swarm.Annotations{
			Labels: map[string]string{
				"com.docker.stack.namespace": "apollo",
			},
			Name: service.Name,
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{
				Image:   service.Image,
				Env:     service.Envs,
				Mounts:  mounts,
				Command: service.Command,
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

	createResp, err := s.dockerClient.ServiceCreate(ctx, spec, serviceCreateOptions())
	if err != nil {
		return "", err
	}
	log.Infof("Swarm Create Resp: %+v", createResp)
	return createResp.ID, nil
}

func serviceCreateOptions() (opts dockerTypes.ServiceCreateOptions) {
	contents, err := ioutil.ReadFile("/run/secrets/stackrox.io/registry_auth")
	if err != nil {
		log.Warnf("Couldn't open registry auth secret: %s", err)
		return
	}
	opts.EncodedRegistryAuth = strings.TrimSpace(string(contents))
	return
}

func (s *swarmOrchestrator) Kill(id string) error {
	return s.dockerClient.ServiceRemove(context.Background(), id)
}

// WaitForCompletion waits for the completion of a global service, by checking the task list
// The RestartPolicy is set to 0
func (s *swarmOrchestrator) WaitForCompletion(name string, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	ticker := time.NewTicker(15 * time.Second)

	for {
		select {
		case <-ticker.C:
			ctx, cancel := docker.TimeoutContext()
			defer cancel()
			f := filters.NewArgs()
			f.Add("name", name)

			tasks, err := s.dockerClient.TaskList(ctx, types.TaskListOptions{Filters: f})
			if err != nil {
				log.Error(err)
				continue
			}
			if len(tasks) == 0 {
				continue
			}
			numNotFinished := len(tasks)
			for _, task := range tasks {
				switch task.Status.State {
				case swarm.TaskStateComplete, swarm.TaskStateShutdown, swarm.TaskStateFailed, swarm.TaskStateRejected:
					numNotFinished--
				}
			}
			if numNotFinished == 0 {
				log.Infof("All tasks are complete for service %v", name)
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("Timed out after %.1f waiting for service %v", timeout.Minutes(), name)
		}
	}
}

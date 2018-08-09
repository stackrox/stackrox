package enforcer

import (
	"context"
	"errors"
	"fmt"
	"time"

	swarmTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	dockerClient "github.com/docker/docker/client"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	logger = logging.LoggerForModule()
)

type enforcer struct {
	*dockerClient.Client
	enforcementMap map[v1.EnforcementAction]enforcers.EnforceFunc
	actionsC       chan *enforcers.DeploymentEnforcement
	stopC          chan struct{}
	stoppedC       chan struct{}
}

// New returns a new Swarm Enforcer.
func New() (enforcers.Enforcer, error) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	dockerClient.NegotiateAPIVersion(ctx)

	e := &enforcer{
		Client:         dockerClient,
		enforcementMap: make(map[v1.EnforcementAction]enforcers.EnforceFunc),
		actionsC:       make(chan *enforcers.DeploymentEnforcement, 10),
		stopC:          make(chan struct{}),
		stoppedC:       make(chan struct{}),
	}
	e.enforcementMap[v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT] = e.scaleToZero
	e.enforcementMap[v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT] = e.unsatisfiableNodeConstraint

	return e, nil
}

func (e *enforcer) Actions() chan<- *enforcers.DeploymentEnforcement {
	return e.actionsC
}

func (e *enforcer) Start() {
	for {
		select {
		case action := <-e.actionsC:
			if f, ok := e.enforcementMap[action.Enforcement]; !ok {
				logger.Errorf("unknown enforcement action: %s", action.Enforcement)
			} else {
				if err := f(action); err != nil {
					logger.Errorf("failed to take enforcement action %s on deployment %s: %s", action.Enforcement, action.Deployment.GetName(), err)
				} else {
					logger.Infof("Successfully taken %s on deployment %s", action.Enforcement, action.Deployment.GetName())
				}
			}
		case <-e.stopC:
			logger.Info("Shutting down swarm Enforcer")
			e.stoppedC <- struct{}{}
		}
	}
}

func (e *enforcer) Stop() {
	e.stopC <- struct{}{}
	<-e.stoppedC
}

func (e *enforcer) scaleToZero(enforcement *enforcers.DeploymentEnforcement) (err error) {
	if len(enforcement.Deployment.GetContainers()) == 0 {
		return errors.New("deployment does not have any containers")
	}

	service, ok := enforcement.OriginalSpec.(swarm.Service)
	if !ok {
		return fmt.Errorf("%+v is not of type swarm service", enforcement.OriginalSpec)
	}
	if service.Spec.Mode.Replicated == nil {
		return fmt.Errorf("service %s is not a replicated service; unable to scale to 0", enforcement.Deployment.GetName())
	}

	service.Spec.Mode.Replicated.Replicas = &[]uint64{0}[0]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = e.ServiceUpdate(ctx, enforcement.Deployment.GetId(), service.Version, service.Spec, swarmTypes.ServiceUpdateOptions{})
	return
}

func (e *enforcer) unsatisfiableNodeConstraint(enforcement *enforcers.DeploymentEnforcement) (err error) {
	service, ok := enforcement.OriginalSpec.(swarm.Service)
	if !ok {
		return fmt.Errorf("%+v is not of type swarm service", enforcement.OriginalSpec)
	}

	task := &service.Spec.TaskTemplate
	if task.Placement == nil {
		task.Placement = &swarm.Placement{}
	}

	placement := task.Placement
	placement.Constraints = append(placement.Constraints, fmt.Sprintf("%s==%s", enforcers.UnsatisfiableNodeConstraintKey, enforcement.AlertID))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = e.ServiceUpdate(ctx, enforcement.Deployment.GetId(), service.Version, service.Spec, swarmTypes.ServiceUpdateOptions{})
	return
}

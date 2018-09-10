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

type enforcerImpl struct {
	*dockerClient.Client
}

// MustCreate creates a new Swarm enforcer or panics.
func MustCreate() enforcers.Enforcer {
	e, err := New()
	if err != nil {
		panic(err)
	}
	return e
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

	e := &enforcerImpl{
		Client: dockerClient,
	}
	enforcementMap := map[v1.EnforcementAction]enforcers.EnforceFunc{
		v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT:                 e.scaleToZero,
		v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT: e.unsatisfiableNodeConstraint,
	}

	return enforcers.CreateEnforcer(enforcementMap), nil
}

func (e *enforcerImpl) scaleToZero(enforcement *enforcers.DeploymentEnforcement) (err error) {
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

func (e *enforcerImpl) unsatisfiableNodeConstraint(enforcement *enforcers.DeploymentEnforcement) (err error) {
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

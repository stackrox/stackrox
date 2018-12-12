package enforcer

import (
	"context"
	"fmt"
	"time"

	swarmTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	dockerClient "github.com/docker/docker/client"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker"
	"github.com/stackrox/rox/pkg/enforcers"
)

// Label key used for unsatisfiable node constraint enforcement.
const (
	UnsatisfiableNodeConstraintKey = `BlockedByStackRoxNext`
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
	dc, err := docker.NewClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	dc.NegotiateAPIVersion(ctx)

	e := &enforcerImpl{
		Client: dc,
	}
	enforcementMap := map[storage.EnforcementAction]enforcers.EnforceFunc{
		storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT:                 e.scaleToZero,
		storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT: e.unsatisfiableNodeConstraint,
		storage.EnforcementAction_KILL_POD_ENFORCEMENT:                      e.kill,
	}

	return enforcers.CreateEnforcer(enforcementMap), nil
}

func (e *enforcerImpl) scaleToZero(enforcement *v1.SensorEnforcement) (err error) {
	deploymentInfo := enforcement.GetDeployment()
	if deploymentInfo == nil {
		return fmt.Errorf("unable to apply constraint to non-deployment")
	}

	service, err := e.loadService(deploymentInfo)
	if err != nil {
		return err
	}

	if service.Spec.Mode.Replicated == nil {
		return fmt.Errorf("service %s is not a replicated service; unable to scale to 0", deploymentInfo.GetDeploymentName())
	}
	service.Spec.Mode.Replicated.Replicas = &[]uint64{0}[0]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = e.ServiceUpdate(ctx, deploymentInfo.GetDeploymentId(), service.Version, service.Spec, swarmTypes.ServiceUpdateOptions{})
	if err != nil {
		return err
	}
	return
}

func (e *enforcerImpl) unsatisfiableNodeConstraint(enforcement *v1.SensorEnforcement) (err error) {
	deploymentInfo := enforcement.GetDeployment()
	if deploymentInfo == nil {
		return fmt.Errorf("unable to apply constraint to non-deployment")
	}

	service, err := e.loadService(deploymentInfo)
	if err != nil {
		return err
	}

	task := &service.Spec.TaskTemplate
	if task.Placement == nil {
		task.Placement = &swarm.Placement{}
	}
	placement := task.Placement
	placement.Constraints = append(placement.Constraints, fmt.Sprintf("%s==%s", UnsatisfiableNodeConstraintKey, deploymentInfo.GetAlertId()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = e.ServiceUpdate(ctx, deploymentInfo.GetDeploymentId(), service.Version, service.Spec, swarmTypes.ServiceUpdateOptions{})
	if err != nil {
		return err
	}
	return
}

func (e *enforcerImpl) kill(enforcement *v1.SensorEnforcement) (err error) {
	containerInfo := enforcement.GetContainerInstance()
	if containerInfo == nil {
		return fmt.Errorf("unable to apply constraint to non-deployment")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err = e.ContainerKill(ctx, containerInfo.GetContainerInstanceId(), "SIGKILL")
	if err != nil {
		return err
	}
	return
}

func (e *enforcerImpl) loadService(deploymentInfo *v1.DeploymentEnforcement) (swarm.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	service, _, err := e.ServiceInspectWithRaw(ctx, deploymentInfo.GetDeploymentId(), swarmTypes.ServiceInspectOptions{})
	return service, err
}

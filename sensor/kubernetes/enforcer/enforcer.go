package enforcer

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/logging"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	logger = logging.LoggerForModule()
)

type enforcer struct {
	client         *kubernetes.Clientset
	enforcementMap map[v1.EnforcementAction]enforcers.EnforceFunc
	actionsC       chan *enforcers.DeploymentEnforcement
	stopC          chan struct{}
	stoppedC       chan struct{}
}

// New returns a new Kubernetes Enforcer.
func New() (enforcers.Enforcer, error) {
	c, err := setupClient()
	if err != nil {
		return nil, err
	}

	e := &enforcer{
		client:         c,
		enforcementMap: make(map[v1.EnforcementAction]enforcers.EnforceFunc),
		actionsC:       make(chan *enforcers.DeploymentEnforcement, 10),
		stopC:          make(chan struct{}),
		stoppedC:       make(chan struct{}),
	}
	e.enforcementMap[v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT] = e.scaleToZero
	e.enforcementMap[v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT] = e.unsatisfiableNodeConstraint

	return e, nil
}

func setupClient() (client *kubernetes.Clientset, err error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return
	}

	return kubernetes.NewForConfig(config)
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
			logger.Info("Shutting down Kubernetes Enforcer")
			e.stoppedC <- struct{}{}
		}
	}
}

func (e *enforcer) Stop() {
	e.stopC <- struct{}{}
	<-e.stoppedC
}

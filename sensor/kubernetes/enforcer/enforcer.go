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

type enforcerImpl struct {
	client *kubernetes.Clientset
}

// MustCreate creates a new enforcer or panics.
func MustCreate() enforcers.Enforcer {
	e, err := New()
	if err != nil {
		panic(err)
	}
	return e
}

// New returns a new Kubernetes Enforcer.
func New() (enforcers.Enforcer, error) {
	c, err := setupClient()
	if err != nil {
		return nil, err
	}

	e := &enforcerImpl{
		client: c,
	}

	enforcementMap := map[v1.EnforcementAction]enforcers.EnforceFunc{
		v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT:                 e.scaleToZero,
		v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT: e.unsatisfiableNodeConstraint,
		v1.EnforcementAction_KILL_POD_ENFORCEMENT:                      e.kill,
	}

	return enforcers.CreateEnforcer(enforcementMap), nil
}

func setupClient() (client *kubernetes.Clientset, err error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return
	}

	return kubernetes.NewForConfig(config)
}

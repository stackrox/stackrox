package enforcer

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/enforcer"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

var (
	log = logging.LoggerForModule()
)

type enforcerImpl struct {
	client   *kubernetes.Clientset
	recorder record.EventRecorder
}

// MustCreate creates a new enforcer or panics.
func MustCreate() enforcer.Enforcer {
	e, err := New()
	if err != nil {
		panic(err)
	}
	return e
}

// New returns a new Kubernetes Enforcer.
func New() (enforcer.Enforcer, error) {
	cl := client.MustCreateClientSet()
	e := &enforcerImpl{
		client:   cl,
		recorder: eventRecorder(cl),
	}

	enforcementMap := map[storage.EnforcementAction]enforcer.EnforceFunc{
		storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT:                 e.scaleToZero,
		storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT: e.unsatisfiableNodeConstraint,
		storage.EnforcementAction_KILL_POD_ENFORCEMENT:                      e.kill,
	}

	return enforcer.CreateEnforcer(enforcementMap), nil
}

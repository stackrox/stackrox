package enforcer

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/enforcer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

var (
	log = logging.LoggerForModule()
)

type enforcerImpl struct {
	client   kubernetes.Interface
	recorder record.EventRecorder
}

// MustCreate creates a new enforcer or panics.
func MustCreate(client kubernetes.Interface) enforcer.Enforcer {
	e, err := New(client)
	if err != nil {
		panic(err)
	}
	return e
}

// New returns a new Kubernetes Enforcer.
func New(cl kubernetes.Interface) (enforcer.Enforcer, error) {
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

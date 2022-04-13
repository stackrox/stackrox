package enforcer

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/sensor/common/enforcer"
	"github.com/stackrox/stackrox/sensor/kubernetes/client"
	"k8s.io/client-go/tools/record"
)

var (
	log = logging.LoggerForModule()
)

type enforcerImpl struct {
	client   client.Interface
	recorder record.EventRecorder
}

// New returns a new Kubernetes Enforcer.
func New(c client.Interface) (enforcer.Enforcer, error) {
	e := &enforcerImpl{
		client:   c,
		recorder: eventRecorder(c.Kubernetes()),
	}

	enforcementMap := map[storage.EnforcementAction]enforcer.EnforceFunc{
		storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT:                 e.scaleToZero,
		storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT: e.unsatisfiableNodeConstraint,
		storage.EnforcementAction_KILL_POD_ENFORCEMENT:                      e.kill,
	}

	return enforcer.CreateEnforcer(enforcementMap), nil
}

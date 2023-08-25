package plan

import (
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	log = logging.LoggerForModule()
)

// ExecutionPlan stores the steps that the upgrader should perform on the K8s API.
type ExecutionPlan struct {
	Creations, Updates []*unstructured.Unstructured
	Deletions          []k8sobjects.ObjectRef
}

// GenerateExecutionPlan generates an execution plan for the given desired state.
func GenerateExecutionPlan(ctx *upgradectx.UpgradeContext, desired []*unstructured.Unstructured, rollback bool) (*ExecutionPlan, error) {
	p := &planner{ctx: ctx, rollback: rollback}
	return p.GenerateExecutionPlan(desired)
}

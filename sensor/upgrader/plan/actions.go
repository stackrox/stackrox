package plan

import (
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ActionName is the name of an action (create, update, delete) in an execution plan.
type ActionName string

// Definition of actions in an execution plan.
const (
	CreateAction ActionName = "create"
	UpdateAction ActionName = "update"
	DeleteAction ActionName = "delete"
)

// ActionDesc describes an action in an ExecutionPlan.
type ActionDesc struct {
	ActionName ActionName // "create", "update", or "delete"
	ObjectRef  k8sobjects.ObjectRef
	Object     *unstructured.Unstructured
}

// Actions returns all actions performed as part of an execution plan, in the correct order (creations, then updates,
// then deletions).
func (p *ExecutionPlan) Actions() []ActionDesc {
	var allActions []ActionDesc
	allActions = append(allActions, actionsForObjects(CreateAction, p.Creations)...)
	allActions = append(allActions, actionsForObjects(UpdateAction, p.Updates)...)
	allActions = append(allActions, actionsForObjectRefs(DeleteAction, p.Deletions)...)
	return allActions
}

func actionsForObjects(actionName ActionName, objects []*unstructured.Unstructured) []ActionDesc {
	descs := make([]ActionDesc, 0, len(objects))
	for _, obj := range objects {
		descs = append(descs, ActionDesc{
			ActionName: actionName,
			ObjectRef:  k8sobjects.RefOf(obj),
			Object:     obj,
		})
	}
	return descs
}

func actionsForObjectRefs(actionName ActionName, objects []k8sobjects.ObjectRef) []ActionDesc {
	descs := make([]ActionDesc, 0, len(objects))
	for _, objRef := range objects {
		descs = append(descs, ActionDesc{
			ActionName: actionName,
			ObjectRef:  objRef,
		})
	}
	return descs
}

// GroupActionsByResource returns all actions in the given slice, grouped by resource.
func GroupActionsByResource(acts []ActionDesc) map[schema.GroupVersionKind][]ActionDesc {
	result := make(map[schema.GroupVersionKind][]ActionDesc)
	for _, act := range acts {
		result[act.ObjectRef.GVK] = append(result[act.ObjectRef.GVK], act)
	}
	return result
}

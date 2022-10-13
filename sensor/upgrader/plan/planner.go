package plan

import (
	"reflect"

	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type planner struct {
	ctx      *upgradectx.UpgradeContext
	rollback bool
}

func (p *planner) objectsAreEqual(a, b *unstructured.Unstructured) bool {
	aCopy := a.DeepCopy()
	bCopy := b.DeepCopy()

	normalizeObject(aCopy)
	normalizeObject(bCopy)

	return reflect.DeepEqual(aCopy.Object, bCopy.Object)
}

func (p *planner) GenerateExecutionPlan(desired []*unstructured.Unstructured) (*ExecutionPlan, error) {
	currObjs, err := p.ctx.ListCurrentObjects()
	if err != nil {
		return nil, err
	}

	currObjMap := k8sobjects.BuildObjectMap(currObjs)

	var plan ExecutionPlan

	for _, desiredObj := range desired {
		ref := k8sobjects.RefOf(desiredObj)
		currObj := currObjMap[ref]

		log.Infof("Testing object %v", ref)
		if currObj == nil {
			plan.Creations = append(plan.Creations, desiredObj)
		} else {
			objectToCreate := desiredObj.DeepCopy()
			if !p.rollback {
				if err := applyPreservedProperties(objectToCreate, currObj); err != nil {
					log.Errorf("Failed to preserve some properties for object %v: %v", ref, err)
				}
			}
			objectsAreEqual := p.objectsAreEqual(currObj, objectToCreate)

			// We don't update if the objects are equal.
			// If the objects are not equal, we check if the object was already modified during this upgrade.
			// If it was, we skip the update.
			// If we're rolling back, though, we DO the update ONLY if it was modified during this upgrade.
			lastModifiedByThisProcessID := currObj.GetAnnotations()[common.LastUpgradeIDAnnotationKey] == p.ctx.ProcessID()
			if !objectsAreEqual && lastModifiedByThisProcessID == p.rollback {
				objectToCreate.SetResourceVersion(currObj.GetResourceVersion())
				plan.Updates = append(plan.Updates, objectToCreate)
			} else {
				log.Infof("Skipping update of object %v as it is unchanged or was already updated. Objects are equal: %v; last modified by this process ID: %v, rollback: %v",
					ref, objectsAreEqual, lastModifiedByThisProcessID, p.rollback)
			}
		}
		delete(currObjMap, ref)
	}

	for remainingObjRef := range currObjMap {
		plan.Deletions = append(plan.Deletions, remainingObjRef)
	}

	// sort objects such that dependency constraints are respected.
	sortObjects(plan.Creations, false)
	sortObjects(plan.Updates, false)
	sortObjectRefs(plan.Deletions, true)

	return &plan, nil
}
